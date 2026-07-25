package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sabai-pos/backend/internal/domain"
	"sabai-pos/backend/internal/store"
)

type SaleService struct {
	store *store.DB
	now   func() time.Time
}

func NewSaleService(st *store.DB) *SaleService {
	return &SaleService{store: st, now: time.Now}
}

type CheckoutLine struct {
	ProductID uuid.UUID
	Qty       int32
}

type CheckoutInput struct {
	Lines         []CheckoutLine
	PaymentMethod string // "cash" | "transfer"
	Paid          int64  // satang (cash); ignored for transfer
	Discount      int64  // satang
	ClientUUID    *uuid.UUID
}

// BillDetail is a bill plus everything needed to render a receipt.
type BillDetail struct {
	Bill        store.Bill
	Items       []store.BillItem
	ShopName    string
	CashierName string
	Voided      bool
}

func billNo(seq int32) string { return fmt.Sprintf("B%06d", seq) }

// Checkout closes a sale: bill + items + stock deduction in one transaction.
// Idempotent on ClientUUID so a retried/offline-synced sale never duplicates.
func (s *SaleService) Checkout(ctx context.Context, storeID uuid.UUID, in CheckoutInput, cashierID uuid.UUID) (BillDetail, error) {
	if len(in.Lines) == 0 {
		return BillDetail{}, domain.Validation("บิลต้องมีสินค้าอย่างน้อย 1 รายการ")
	}
	if in.PaymentMethod != "cash" && in.PaymentMethod != "transfer" {
		return BillDetail{}, domain.Validation("วิธีชำระเงินไม่ถูกต้อง")
	}
	if in.Discount < 0 {
		return BillDetail{}, domain.Validation("ส่วนลดต้องไม่ติดลบ")
	}

	// Idempotency short-circuit.
	if in.ClientUUID != nil {
		if existing, err := s.store.GetBillByClientUUID(ctx, store.GetBillByClientUUIDParams{
			StoreID: storeID, ClientUuid: *in.ClientUUID,
		}); err == nil {
			return s.assembleDetail(ctx, storeID, existing)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return BillDetail{}, err
		}
	}

	clientUUID := uuid.New()
	if in.ClientUUID != nil {
		clientUUID = *in.ClientUUID
	}

	var bill store.Bill
	var items []store.BillItem
	err := s.store.ExecTx(ctx, func(q *store.Queries) error {
		// Price + stock, locking each inventory row against oversell.
		var subtotal int64
		type staged struct {
			pid       uuid.UUID
			name      string
			price     int64
			qty       int32
			lineTotal int64
		}
		stagedLines := make([]staged, 0, len(in.Lines))
		for _, ln := range in.Lines {
			if ln.Qty <= 0 {
				return domain.Validation("จำนวนสินค้าต้องมากกว่า 0")
			}
			p, err := q.GetProduct(ctx, store.GetProductParams{ID: ln.ProductID, StoreID: storeID})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return domain.NotFound("ไม่พบสินค้าในบิล")
				}
				return err
			}
			if !p.IsActive {
				return domain.Validation("สินค้า \"" + p.Name + "\" ถูกปิดการขาย")
			}
			inv, err := q.GetInventoryForUpdate(ctx, ln.ProductID)
			if err != nil {
				return err
			}
			if inv.QtyOnHand < ln.Qty {
				return domain.Validation(fmt.Sprintf("สต็อกไม่พอ: %s เหลือ %d ต้องการ %d", p.Name, inv.QtyOnHand, ln.Qty))
			}
			lineTotal := p.SellPrice * int64(ln.Qty)
			subtotal += lineTotal
			stagedLines = append(stagedLines, staged{ln.ProductID, p.Name, p.SellPrice, ln.Qty, lineTotal})
		}

		total := subtotal - in.Discount
		if total < 0 {
			return domain.Validation("ส่วนลดมากกว่ายอดรวม")
		}
		paid := in.Paid
		if in.PaymentMethod == "transfer" {
			paid = total
		}
		if paid < total {
			return domain.Validation("รับเงินไม่พอกับยอดที่ต้องชำระ")
		}
		change := paid - total

		seq, err := q.NextBillSeq(ctx, storeID)
		if err != nil {
			return err
		}
		bill, err = q.CreateBill(ctx, store.CreateBillParams{
			StoreID:       storeID,
			BillNo:        billNo(seq),
			ClientUuid:    clientUUID,
			CashierID:     store.PgUUID(cashierID),
			Subtotal:      subtotal,
			Discount:      in.Discount,
			Total:         total,
			Paid:          paid,
			Change:        change,
			PaymentMethod: in.PaymentMethod,
			Status:        "completed",
			SyncedAt:      store.PgTime(s.now()),
		})
		if err != nil {
			return err
		}

		refType := "bill"
		for _, sl := range stagedLines {
			item, err := q.CreateBillItem(ctx, store.CreateBillItemParams{
				BillID:        bill.ID,
				ProductID:     store.PgUUID(sl.pid),
				NameSnapshot:  sl.name,
				PriceSnapshot: sl.price,
				Qty:           sl.qty,
				LineTotal:     sl.lineTotal,
			})
			if err != nil {
				return err
			}
			items = append(items, item)
			if _, err := q.CreateMovement(ctx, store.CreateMovementParams{
				StoreID:   storeID,
				ProductID: sl.pid,
				Type:      "sale",
				QtyDelta:  -sl.qty,
				RefType:   &refType,
				RefID:     store.PgUUID(bill.ID),
				CreatedBy: store.PgUUID(cashierID),
			}); err != nil {
				return err
			}
			if _, err := q.AddInventoryQty(ctx, store.AddInventoryQtyParams{ProductID: sl.pid, QtyOnHand: -sl.qty}); err != nil {
				return err
			}
		}
		return audit(ctx, q, storeID, cashierID, "sale", "bill", bill.ID, map[string]any{
			"bill_no": bill.BillNo, "total": bill.Total, "method": bill.PaymentMethod,
		})
	})
	if err != nil {
		// Concurrent duplicate of the same client_uuid → return the winner.
		if isUniqueViolation(err) && in.ClientUUID != nil {
			if existing, e := s.store.GetBillByClientUUID(ctx, store.GetBillByClientUUIDParams{StoreID: storeID, ClientUuid: *in.ClientUUID}); e == nil {
				return s.assembleDetail(ctx, storeID, existing)
			}
		}
		return BillDetail{}, err
	}
	return s.detailFrom(ctx, storeID, bill, items), nil
}

// Void reverses a completed bill by creating a new void bill that references it
// (the original stays immutable) and restoring stock through the ledger.
func (s *SaleService) Void(ctx context.Context, storeID, billID uuid.UUID, reason string, actorID uuid.UUID) (store.Bill, error) {
	var voidBill store.Bill
	err := s.store.ExecTx(ctx, func(q *store.Queries) error {
		orig, err := q.GetBill(ctx, store.GetBillParams{ID: billID, StoreID: storeID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.NotFound("ไม่พบบิล")
			}
			return err
		}
		if orig.Status != "completed" {
			return domain.Conflict("บิลนี้ยกเลิกไม่ได้ (สถานะไม่ใช่ completed)")
		}
		if _, err := q.GetVoidForBill(ctx, store.GetVoidForBillParams{StoreID: storeID, VoidsBillID: store.PgUUID(orig.ID)}); err == nil {
			return domain.Conflict("บิลนี้ถูกยกเลิกไปแล้ว")
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		items, err := q.ListBillItems(ctx, orig.ID)
		if err != nil {
			return err
		}
		seq, err := q.NextBillSeq(ctx, storeID)
		if err != nil {
			return err
		}
		voidBill, err = q.CreateBill(ctx, store.CreateBillParams{
			StoreID:       storeID,
			BillNo:        billNo(seq),
			ClientUuid:    uuid.New(),
			CashierID:     store.PgUUID(actorID),
			Subtotal:      orig.Subtotal,
			Discount:      orig.Discount,
			Total:         orig.Total,
			Paid:          0,
			Change:        0,
			PaymentMethod: orig.PaymentMethod,
			Status:        "void",
			VoidsBillID:   store.PgUUID(orig.ID),
			SyncedAt:      store.PgTime(s.now()),
		})
		if err != nil {
			return err
		}
		refType := "void"
		for _, it := range items {
			if _, err := q.CreateBillItem(ctx, store.CreateBillItemParams{
				BillID:        voidBill.ID,
				ProductID:     it.ProductID,
				NameSnapshot:  it.NameSnapshot,
				PriceSnapshot: it.PriceSnapshot,
				Qty:           it.Qty,
				LineTotal:     it.LineTotal,
			}); err != nil {
				return err
			}
			pid := uuid.UUID(it.ProductID.Bytes)
			if _, err := q.CreateMovement(ctx, store.CreateMovementParams{
				StoreID:   storeID,
				ProductID: pid,
				Type:      "void",
				QtyDelta:  it.Qty, // restore stock
				RefType:   &refType,
				RefID:     store.PgUUID(voidBill.ID),
				Reason:    strPtr(reason),
				CreatedBy: store.PgUUID(actorID),
			}); err != nil {
				return err
			}
			if _, err := q.AddInventoryQty(ctx, store.AddInventoryQtyParams{ProductID: pid, QtyOnHand: it.Qty}); err != nil {
				return err
			}
		}
		return audit(ctx, q, storeID, actorID, "void", "bill", orig.ID, map[string]any{
			"void_bill_no": voidBill.BillNo, "orig_bill_no": orig.BillNo, "reason": reason,
		})
	})
	if err != nil {
		return store.Bill{}, err
	}
	return voidBill, nil
}

func (s *SaleService) GetBillDetail(ctx context.Context, storeID, id uuid.UUID) (BillDetail, error) {
	bill, err := s.store.GetBill(ctx, store.GetBillParams{ID: id, StoreID: storeID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BillDetail{}, domain.NotFound("ไม่พบบิล")
		}
		return BillDetail{}, err
	}
	return s.assembleDetail(ctx, storeID, bill)
}

func (s *SaleService) assembleDetail(ctx context.Context, storeID uuid.UUID, bill store.Bill) (BillDetail, error) {
	items, err := s.store.ListBillItems(ctx, bill.ID)
	if err != nil {
		return BillDetail{}, err
	}
	return s.detailFrom(ctx, storeID, bill, items), nil
}

func (s *SaleService) detailFrom(ctx context.Context, storeID uuid.UUID, bill store.Bill, items []store.BillItem) BillDetail {
	d := BillDetail{Bill: bill, Items: items}
	if st, err := s.store.GetStore(ctx, storeID); err == nil {
		d.ShopName = st.Name
	}
	if bill.CashierID.Valid {
		if u, err := s.store.GetUserByID(ctx, uuid.UUID(bill.CashierID.Bytes)); err == nil {
			d.CashierName = u.Username
		}
	}
	if v, err := s.store.GetVoidForBill(ctx, store.GetVoidForBillParams{StoreID: storeID, VoidsBillID: store.PgUUID(bill.ID)}); err == nil && v.ID != uuid.Nil {
		d.Voided = true
	}
	return d
}

// audit writes an entry to audit_log within the caller's transaction.
func audit(ctx context.Context, q *store.Queries, storeID, actor uuid.UUID, action, entity string, entityID uuid.UUID, detail map[string]any) error {
	raw, _ := json.Marshal(detail)
	ent := entity
	return q.CreateAuditLog(ctx, store.CreateAuditLogParams{
		StoreID:  storeID,
		ActorID:  store.PgUUID(actor),
		Action:   action,
		Entity:   &ent,
		EntityID: store.PgUUID(entityID),
		Detail:   raw,
	})
}
