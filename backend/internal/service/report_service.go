package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sabai-pos/backend/internal/store"
)

type ReportService struct {
	store *store.DB
	now   func() time.Time
}

func NewReportService(st *store.DB) *ReportService {
	return &ReportService{store: st, now: time.Now}
}

func bangkok() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*3600) // +07:00 fallback
	}
	return loc
}

type Summary struct {
	SalesToday int64 `json:"sales_today"` // satang
	Bills      int64 `json:"bills"`
	Profit     int64 `json:"profit"` // satang, approximate
	MarginPct  int   `json:"margin_pct"`
	AvgPerBill int64 `json:"avg_per_bill"` // satang
	LowStock   int64 `json:"low_stock"`
}

func (s *ReportService) Summary(ctx context.Context, storeID uuid.UUID) (Summary, error) {
	loc := bangkok()
	now := s.now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	totals, err := s.store.SalesTotals(ctx, store.SalesTotalsParams{
		StoreID: storeID, CreatedAt: store.PgTime(start), CreatedAt_2: store.PgTime(end),
	})
	if err != nil {
		return Summary{}, err
	}
	profit, err := s.store.ProfitApprox(ctx, store.ProfitApproxParams{
		StoreID: storeID, CreatedAt: store.PgTime(start), CreatedAt_2: store.PgTime(end),
	})
	if err != nil {
		return Summary{}, err
	}
	low, err := s.store.LowStockCount(ctx, storeID)
	if err != nil {
		return Summary{}, err
	}
	out := Summary{SalesToday: totals.Sales, Bills: totals.Bills, Profit: profit, LowStock: low}
	if totals.Sales > 0 {
		out.MarginPct = int(profit * 100 / totals.Sales)
	}
	if totals.Bills > 0 {
		out.AvgPerBill = totals.Sales / totals.Bills
	}
	return out, nil
}

type TopProduct struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	QtySold int64  `json:"qty_sold"`
}

func (s *ReportService) Top(ctx context.Context, storeID uuid.UUID, limit int32) ([]TopProduct, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	rows, err := s.store.TopProducts(ctx, store.TopProductsParams{StoreID: storeID, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]TopProduct, 0, len(rows))
	for _, r := range rows {
		out = append(out, TopProduct{ID: r.ID.String(), Name: r.Name, QtySold: r.QtySold})
	}
	return out, nil
}

type DaySales struct {
	Day   string `json:"day"` // YYYY-MM-DD (Bangkok)
	Sales int64  `json:"sales"`
}

func (s *ReportService) Daily(ctx context.Context, storeID uuid.UUID, days int) ([]DaySales, error) {
	if days <= 0 || days > 90 {
		days = 7
	}
	loc := bangkok()
	now := s.now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))

	rows, err := s.store.DailySales(ctx, store.DailySalesParams{StoreID: storeID, CreatedAt: store.PgTime(start)})
	if err != nil {
		return nil, err
	}
	byDay := make(map[string]int64, len(rows))
	for _, r := range rows {
		byDay[r.Day.Time.Format("2006-01-02")] = r.Sales
	}
	// Fill every day in the window (0 for days with no sales).
	out := make([]DaySales, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		out = append(out, DaySales{Day: d, Sales: byDay[d]})
	}
	return out, nil
}
