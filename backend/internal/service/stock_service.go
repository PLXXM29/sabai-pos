package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"minimart-pos/backend/internal/domain"
	"minimart-pos/backend/internal/store"
)

type StockService struct {
	store *store.DB
}

func NewStockService(st *store.DB) *StockService {
	return &StockService{store: st}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Receive records a stock intake: one append-only ledger movement plus the
// cached on-hand update, atomically. Idempotent on clientUUID so a retried
// offline sync never double-counts.
func (s *StockService) Receive(ctx context.Context, storeID, productID uuid.UUID, qty int32, reason string, actor uuid.UUID, clientUUID *uuid.UUID) (store.Inventory, error) {
	if qty <= 0 {
		return store.Inventory{}, domain.Validation("จำนวนที่รับต้องมากกว่า 0")
	}
	// Ensure the product exists in this store (also blocks cross-store writes).
	if _, err := s.store.GetProduct(ctx, store.GetProductParams{ID: productID, StoreID: storeID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Inventory{}, domain.NotFound("ไม่พบสินค้า")
		}
		return store.Inventory{}, err
	}
	// Idempotency: if this client_uuid was already applied, return current state.
	if clientUUID != nil {
		if _, err := s.store.GetMovementByClientUUID(ctx, store.GetMovementByClientUUIDParams{
			StoreID:    storeID,
			ClientUuid: store.PgUUID(*clientUUID),
		}); err == nil {
			return s.store.GetInventory(ctx, productID)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return store.Inventory{}, err
		}
	}

	var out store.Inventory
	err := s.store.ExecTx(ctx, func(q *store.Queries) error {
		if _, err := q.CreateMovement(ctx, store.CreateMovementParams{
			StoreID:    storeID,
			ProductID:  productID,
			Type:       "receive",
			QtyDelta:   qty,
			Reason:     strPtr(reason),
			CreatedBy:  store.PgUUID(actor),
			ClientUuid: store.PgUUIDPtr(clientUUID),
		}); err != nil {
			return err
		}
		inv, err := q.AddInventoryQty(ctx, store.AddInventoryQtyParams{ProductID: productID, QtyOnHand: qty})
		if err != nil {
			return err
		}
		out = inv
		return nil
	})
	if err != nil {
		// Concurrent duplicate of the same client_uuid → treat as already applied.
		if isUniqueViolation(err) {
			return s.store.GetInventory(ctx, productID)
		}
		return store.Inventory{}, err
	}
	return out, nil
}

// OnHand returns the authoritative on-hand computed from the immutable ledger.
func (s *StockService) OnHand(ctx context.Context, productID uuid.UUID) (int64, error) {
	return s.store.OnHandFromLedger(ctx, productID)
}
