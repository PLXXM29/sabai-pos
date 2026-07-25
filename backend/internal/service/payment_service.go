package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sabai-pos/backend/internal/domain"
	"sabai-pos/backend/internal/store"
)

const paymentTTL = 10 * time.Minute

type PaymentService struct {
	store *store.DB
	now   func() time.Time
}

func NewPaymentService(st *store.DB) *PaymentService {
	return &PaymentService{store: st, now: time.Now}
}

// Create opens a pending payment intent that the notify webhook will match.
func (s *PaymentService) Create(ctx context.Context, storeID uuid.UUID, amount int64, billUUID *uuid.UUID) (store.Payment, error) {
	if amount <= 0 {
		return store.Payment{}, domain.Validation("ยอดชำระต้องมากกว่า 0")
	}
	return s.store.CreatePayment(ctx, store.CreatePaymentParams{
		StoreID:        storeID,
		BillClientUuid: store.PgUUIDPtr(billUUID),
		Amount:         amount,
		ExpiresAt:      store.PgTime(s.now().Add(paymentTTL)),
	})
}

func (s *PaymentService) Get(ctx context.Context, storeID, id uuid.UUID) (store.Payment, error) {
	p, err := s.store.GetPayment(ctx, store.GetPaymentParams{ID: id, StoreID: storeID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Payment{}, domain.NotFound("ไม่พบรายการชำระเงิน")
		}
		return store.Payment{}, err
	}
	return p, nil
}

func (s *PaymentService) Cancel(ctx context.Context, storeID, id uuid.UUID) error {
	return s.store.CancelPayment(ctx, store.CancelPaymentParams{ID: id, StoreID: storeID})
}

// Notify matches a bank "money received" amount to the oldest pending intent and
// marks it paid. Returns whether a match was found.
func (s *PaymentService) Notify(ctx context.Context, amount int64, ref, note string) (bool, error) {
	matched := false
	err := s.store.ExecTx(ctx, func(q *store.Queries) error {
		p, err := q.MatchPendingPaymentForUpdate(ctx, amount)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // no pending intent for this amount — ignore
			}
			return err
		}
		if err := q.MarkPaymentPaid(ctx, store.MarkPaymentPaidParams{
			ID: p.ID, Ref: strPtr(ref), RawNote: strPtr(note),
		}); err != nil {
			return err
		}
		matched = true
		return nil
	})
	return matched, err
}
