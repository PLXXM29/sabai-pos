package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"minimart-pos/backend/internal/domain"
	"minimart-pos/backend/internal/store"
)

type CatalogService struct {
	store *store.DB
}

func NewCatalogService(st *store.DB) *CatalogService {
	return &CatalogService{store: st}
}

type ProductInput struct {
	Name         string
	Barcode      *string
	Category     string
	CostPrice    int64 // satang
	SellPrice    int64 // satang
	ReorderPoint int32
}

func (in *ProductInput) validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return domain.Validation("ต้องระบุชื่อสินค้า")
	}
	if in.SellPrice <= 0 {
		return domain.Validation("ราคาขายต้องมากกว่า 0")
	}
	if in.CostPrice < 0 {
		return domain.Validation("ราคาทุนต้องไม่ติดลบ")
	}
	if in.ReorderPoint < 0 {
		return domain.Validation("จุดสั่งซื้อต้องไม่ติดลบ")
	}
	return nil
}

func normBarcode(b *string) *string {
	if b == nil {
		return nil
	}
	t := strings.TrimSpace(*b)
	if t == "" {
		return nil
	}
	return &t
}

// Create inserts a product and its inventory row atomically.
func (s *CatalogService) Create(ctx context.Context, storeID uuid.UUID, in ProductInput) (store.Product, error) {
	if err := in.validate(); err != nil {
		return store.Product{}, err
	}
	var out store.Product
	err := s.store.ExecTx(ctx, func(q *store.Queries) error {
		p, err := q.CreateProduct(ctx, store.CreateProductParams{
			StoreID:   storeID,
			Name:      strings.TrimSpace(in.Name),
			Barcode:   normBarcode(in.Barcode),
			Category:  in.Category,
			CostPrice: in.CostPrice,
			SellPrice: in.SellPrice,
		})
		if err != nil {
			return err
		}
		if _, err := q.CreateInventory(ctx, store.CreateInventoryParams{
			ProductID:    p.ID,
			StoreID:      storeID,
			QtyOnHand:    0,
			ReorderPoint: in.ReorderPoint,
		}); err != nil {
			return err
		}
		out = p
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return store.Product{}, domain.Conflict("บาร์โค้ดนี้มีอยู่แล้ว")
		}
		return store.Product{}, err
	}
	return out, nil
}

func (s *CatalogService) Update(ctx context.Context, storeID, id uuid.UUID, in ProductInput) (store.Product, error) {
	if err := in.validate(); err != nil {
		return store.Product{}, err
	}
	p, err := s.store.UpdateProduct(ctx, store.UpdateProductParams{
		ID:        id,
		StoreID:   storeID,
		Name:      strings.TrimSpace(in.Name),
		Barcode:   normBarcode(in.Barcode),
		Category:  in.Category,
		CostPrice: in.CostPrice,
		SellPrice: in.SellPrice,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Product{}, domain.NotFound("ไม่พบสินค้า")
		}
		if isUniqueViolation(err) {
			return store.Product{}, domain.Conflict("บาร์โค้ดนี้มีอยู่แล้ว")
		}
		return store.Product{}, err
	}
	return p, nil
}

func (s *CatalogService) List(ctx context.Context, storeID uuid.UUID) ([]store.ListProductsWithStockRow, error) {
	return s.store.ListProductsWithStock(ctx, storeID)
}

func (s *CatalogService) Get(ctx context.Context, storeID, id uuid.UUID) (store.Product, error) {
	p, err := s.store.GetProduct(ctx, store.GetProductParams{ID: id, StoreID: storeID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Product{}, domain.NotFound("ไม่พบสินค้า")
		}
		return store.Product{}, err
	}
	return p, nil
}

func (s *CatalogService) Deactivate(ctx context.Context, storeID, id uuid.UUID) error {
	n, err := s.store.DeactivateProduct(ctx, store.DeactivateProductParams{ID: id, StoreID: storeID})
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.NotFound("ไม่พบสินค้า")
	}
	return nil
}
