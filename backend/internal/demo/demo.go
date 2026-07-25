// Package demo builds and maintains the sample dataset behind the public demo.
//
// A showcase deployment has two needs a plain fixture cannot meet. It must look
// like a shop that has been trading for a month — otherwise every report reads
// zero and the interesting screens look broken — and it must be able to heal
// itself, because visitors will (and should) delete products and void bills.
//
// So the dataset is generated rather than dumped: a deterministic pseudo-random
// month of trading, replayable from scratch at any time. Deterministic matters
// because a demo that shows different numbers on every reset is impossible to
// write documentation, or a portfolio case study, against.
//
// The generated history obeys the same invariants the application enforces at
// runtime — money is integer satang, bill numbers are gap-free, and stock is
// derived from the append-only movement ledger — so nothing in here can leave
// the database in a state the app itself would refuse to create.
package demo

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"sabai-pos/backend/internal/auth"
)

// DefaultHistoryDays is a month of trading: long enough for the 7-day and
// 30-day report ranges to differ, short enough to rebuild in a second or two.
const DefaultHistoryDays = 30

// bangkok is the shop's wall clock. Reports bucket by Bangkok calendar day, so
// the generated timestamps have to land in the right local day or "yesterday"
// drifts. Thailand has had a fixed +07:00 offset with no DST since 1976, which
// makes the fallback exact rather than approximate — it only matters for a
// binary built without the tzdata table.
var bangkok = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Bangkok"); err == nil {
		return loc
	}
	return time.FixedZone("ICT", 7*60*60)
}()

// seed fixes the pseudo-random stream. Any constant works; this one is the date
// the demo dataset was designed.
const randSeed = 20260725

type Seeder struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func New(pool *pgxpool.Pool) *Seeder {
	return &Seeder{pool: pool, now: time.Now}
}

// Result summarises what a build produced, for logging and for the reset
// endpoint's response.
type Result struct {
	StoreID     uuid.UUID `json:"store_id"`
	ShopName    string    `json:"shop_name"`
	Users       int       `json:"users"`
	Products    int       `json:"products"`
	Bills       int       `json:"bills"`
	Items       int       `json:"items"`
	HistoryDays int       `json:"history_days"`
	Took        string    `json:"took"`
}

// Ensure builds the dataset if and only if the database holds no store yet, and
// reports whether it did. Safe to call on every boot: an existing demo (with
// whatever visitors have done to it since) is left alone.
func (s *Seeder) Ensure(ctx context.Context, historyDays int) (Result, bool, error) {
	var stores int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM stores`).Scan(&stores); err != nil {
		return Result{}, false, fmt.Errorf("count stores: %w", err)
	}
	if stores > 0 {
		return Result{}, false, nil
	}
	res, err := s.Reset(ctx, historyDays)
	return res, err == nil, err
}

// RefreshIfStale rebuilds the dataset when the last build is older than maxAge,
// and reports whether it did.
//
// Age is measured from a timestamp in the database rather than from a ticker in
// this process, because the process is not around long enough to be the clock:
// the app scales to zero between visitors, so an in-memory 24-hour timer would
// almost never fire, and the promise of a daily rebuild would quietly not hold.
// Called on boot and on a short interval while running.
func (s *Seeder) RefreshIfStale(ctx context.Context, maxAge time.Duration, historyDays int) (Result, bool, error) {
	if maxAge <= 0 {
		return Result{}, false, nil
	}

	var seededAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT (config->>'demo_seeded_at')::timestamptz FROM stores ORDER BY created_at LIMIT 1`,
	).Scan(&seededAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Result{}, false, nil // nothing seeded yet — Ensure handles that
		}
		return Result{}, false, fmt.Errorf("read demo age: %w", err)
	}
	// A nil timestamp means the data was seeded by a build that predates this
	// bookkeeping; treat it as stale so it adopts the convention on this pass.
	if seededAt != nil && s.now().Sub(*seededAt) < maxAge {
		return Result{}, false, nil
	}

	res, err := s.Reset(ctx, historyDays)
	return res, err == nil, err
}

// Reset discards all business data and rebuilds the sample dataset from scratch.
//
// Everything happens in one transaction, so a visitor mid-sale during a reset
// either sees the old dataset or the new one — never half of each.
func (s *Seeder) Reset(ctx context.Context, historyDays int) (Result, error) {
	if historyDays < 0 {
		historyDays = 0
	}
	started := time.Now()

	// Argon2id is intentionally slow. Hash outside the transaction so the write
	// lock is held for as short a time as possible.
	hashes := make([]string, len(Accounts))
	for i, a := range Accounts {
		h, err := auth.HashPassword(a.Password)
		if err != nil {
			return Result{}, fmt.Errorf("hash %s: %w", a.Username, err)
		}
		hashes[i] = h
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once committed

	// Every business table hangs off stores(id) — that is the multi-tenancy
	// invariant the schema is built on — so one CASCADE reaches all of them and
	// cannot be left behind by a future table the way an explicit list can.
	// TRUNCATE (not DELETE) is also what makes this possible at all: bills,
	// bill_items and stock_movements carry triggers that reject DELETE.
	if _, err := tx.Exec(ctx, `TRUNCATE TABLE stores CASCADE`); err != nil {
		return Result{}, fmt.Errorf("wipe: %w", err)
	}

	// The build timestamp lives in the database, not in process memory: the
	// deployment scales to zero, so "how long since the last rebuild" cannot be
	// answered by anything this process is holding. See RefreshIfStale.
	storeID := uuid.New()
	if _, err := tx.Exec(ctx,
		`INSERT INTO stores (id, name, config)
		 VALUES ($1, $2, jsonb_build_object('demo_seeded_at', to_jsonb(now())))`,
		pgUUID(storeID), shopName,
	); err != nil {
		return Result{}, fmt.Errorf("create store: %w", err)
	}

	userIDs := make([]uuid.UUID, len(Accounts))
	for i, a := range Accounts {
		userIDs[i] = uuid.New()
		if _, err := tx.Exec(ctx,
			`INSERT INTO users (id, store_id, username, password_hash, role)
			 VALUES ($1, $2, $3, $4, $5)`,
			pgUUID(userIDs[i]), pgUUID(storeID), a.Username, hashes[i], a.Role,
		); err != nil {
			return Result{}, fmt.Errorf("create user %s: %w", a.Username, err)
		}
	}

	productIDs := make([]uuid.UUID, len(catalog))
	for i, p := range catalog {
		productIDs[i] = uuid.New()
		bc := barcode(p.id)
		if _, err := tx.Exec(ctx,
			`INSERT INTO products (id, store_id, name, barcode, category, cost_price, sell_price)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			pgUUID(productIDs[i]), pgUUID(storeID), p.name, bc, p.category, p.cost, p.price,
		); err != nil {
			return Result{}, fmt.Errorf("create product %q: %w", p.name, err)
		}
	}

	sales, voids := plan(s.now().In(bangkok), historyDays)

	// Net units leaving the shelf: a voided bill puts its items back, so it must
	// not inflate the opening stock.
	sold := make([]int32, len(catalog))
	for _, b := range sales {
		for _, ln := range b.lines {
			sold[ln.product] += ln.qty
		}
	}
	for _, v := range voids {
		for _, ln := range sales[v.of].lines {
			sold[ln.product] -= ln.qty
		}
	}

	// Opening stock is receipted the day before trading starts, for exactly what
	// the month consumed plus the on-hand figure the demo should end on. That is
	// what keeps inventory.qty_on_hand equal to the sum of the ledger.
	openedAt := s.now().In(bangkok).AddDate(0, 0, -(historyDays + 1)).Truncate(time.Hour)
	openingReason := "สต็อกยกมาตอนเปิดร้าน"

	movements := make([][]any, 0, len(catalog)+len(sales)*3)
	for i, p := range catalog {
		opening := sold[i] + p.onHand
		if _, err := tx.Exec(ctx,
			`INSERT INTO inventory (product_id, store_id, qty_on_hand, reorder_point)
			 VALUES ($1, $2, $3, $4)`,
			pgUUID(productIDs[i]), pgUUID(storeID), p.onHand, p.reorder,
		); err != nil {
			return Result{}, fmt.Errorf("create inventory %q: %w", p.name, err)
		}
		if opening > 0 {
			movements = append(movements, []any{
				pgUUID(uuid.New()), pgUUID(storeID), pgUUID(productIDs[i]),
				"receive", opening, nil, pgtype.UUID{}, &openingReason,
				pgUUID(userIDs[0]), pgtype.UUID{}, openedAt,
			})
		}
	}

	// Bill numbers are assigned in chronological order across sales *and* voids,
	// because a void is itself a numbered document in this schema.
	type numbered struct {
		at    time.Time
		sale  *genBill
		voidO *genVoid
	}
	ordered := make([]numbered, 0, len(sales)+len(voids))
	for i := range sales {
		ordered = append(ordered, numbered{at: sales[i].at, sale: &sales[i]})
	}
	for i := range voids {
		ordered = append(ordered, numbered{at: voids[i].at, voidO: &voids[i]})
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].at.Before(ordered[j].at) })

	billRows := make([][]any, 0, len(ordered))
	itemRows := make([][]any, 0, len(sales)*3)
	refBill, refVoid := "bill", "void"

	for i, o := range ordered {
		no := fmt.Sprintf("B%06d", i+1)

		if b := o.sale; b != nil {
			var subtotal int64
			for _, ln := range b.lines {
				subtotal += catalog[ln.product].price * int64(ln.qty)
			}
			total := subtotal - b.discount
			paid, change := total, int64(0)
			if b.method == "cash" {
				paid = b.paid
				change = paid - total
			}
			billRows = append(billRows, []any{
				pgUUID(b.id), pgUUID(storeID), no, pgUUID(uuid.New()), pgUUID(userIDs[b.cashier]),
				subtotal, b.discount, total, paid, change, b.method, "completed",
				pgtype.UUID{}, b.at, b.at,
			})
			for _, ln := range b.lines {
				p := catalog[ln.product]
				itemRows = append(itemRows, []any{
					pgUUID(uuid.New()), pgUUID(b.id), pgUUID(productIDs[ln.product]),
					p.name, p.price, ln.qty, p.price * int64(ln.qty),
				})
				movements = append(movements, []any{
					pgUUID(uuid.New()), pgUUID(storeID), pgUUID(productIDs[ln.product]),
					"sale", -ln.qty, &refBill, pgUUID(b.id), nil,
					pgUUID(userIDs[b.cashier]), pgtype.UUID{}, b.at,
				})
			}
			continue
		}

		// A void does not edit the original bill: it is a new document that
		// mirrors it, references it, and returns the stock through the ledger.
		v := o.voidO
		orig := &sales[v.of]
		var subtotal int64
		for _, ln := range orig.lines {
			subtotal += catalog[ln.product].price * int64(ln.qty)
		}
		billRows = append(billRows, []any{
			pgUUID(v.id), pgUUID(storeID), no, pgUUID(uuid.New()), pgUUID(userIDs[1]),
			subtotal, orig.discount, subtotal - orig.discount, int64(0), int64(0),
			orig.method, "void", pgUUID(orig.id), v.at, v.at,
		})
		for _, ln := range orig.lines {
			p := catalog[ln.product]
			itemRows = append(itemRows, []any{
				pgUUID(uuid.New()), pgUUID(v.id), pgUUID(productIDs[ln.product]),
				p.name, p.price, ln.qty, p.price * int64(ln.qty),
			})
			movements = append(movements, []any{
				pgUUID(uuid.New()), pgUUID(storeID), pgUUID(productIDs[ln.product]),
				"void", ln.qty, &refVoid, pgUUID(v.id), &v.reason,
				pgUUID(userIDs[1]), pgtype.UUID{}, v.at,
			})
		}
	}

	if err := copyRows(ctx, tx, "bills",
		[]string{"id", "store_id", "bill_no", "client_uuid", "cashier_id",
			"subtotal", "discount", "total", "paid", "change",
			"payment_method", "status", "voids_bill_id", "created_at", "synced_at"},
		billRows); err != nil {
		return Result{}, err
	}
	if err := copyRows(ctx, tx, "bill_items",
		[]string{"id", "bill_id", "product_id", "name_snapshot", "price_snapshot", "qty", "line_total"},
		itemRows); err != nil {
		return Result{}, err
	}
	if err := copyRows(ctx, tx, "stock_movements",
		[]string{"id", "store_id", "product_id", "type", "qty_delta",
			"ref_type", "ref_id", "reason", "created_by", "client_uuid", "created_at"},
		movements); err != nil {
		return Result{}, err
	}

	// Leave the counter where the history stopped, so the next real sale
	// continues the sequence instead of colliding with it.
	if _, err := tx.Exec(ctx,
		`INSERT INTO bill_counters (store_id, next_seq) VALUES ($1, $2)`,
		pgUUID(storeID), int64(len(ordered)+1),
	); err != nil {
		return Result{}, fmt.Errorf("set bill counter: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit: %w", err)
	}

	return Result{
		StoreID:     storeID,
		ShopName:    shopName,
		Users:       len(Accounts),
		Products:    len(catalog),
		Bills:       len(billRows),
		Items:       len(itemRows),
		HistoryDays: historyDays,
		Took:        time.Since(started).Round(time.Millisecond).String(),
	}, nil
}

func copyRows(ctx context.Context, tx pgx.Tx, table string, cols []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("load %s: %w", table, err)
	}
	return nil
}

func barcode(id int) string { return fmt.Sprintf("885%d", 1000000000+id*7919) }

func pgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }

// ── history generator ───────────────────────────────────────────────────────

type genLine struct {
	product int // index into catalog
	qty     int32
}

type genBill struct {
	id       uuid.UUID
	at       time.Time
	lines    []genLine
	method   string // cash | transfer
	discount int64  // satang
	paid     int64  // satang, cash only
	cashier  int    // index into Accounts
}

type genVoid struct {
	id     uuid.UUID
	at     time.Time
	of     int // index into the sales slice
	reason string
}

// plan invents a month of trading. Shape, not noise: mornings and the
// after-work evening are busy, weekends beat weekdays, most baskets are two or
// three items, and a handful of bills get voided the way they do in a real shop.
func plan(now time.Time, days int) ([]genBill, []genVoid) {
	if days == 0 {
		return nil, nil
	}
	rng := rand.New(rand.NewSource(randSeed))

	// Relative volume per hour of the day, 06:00 through 21:00.
	hourWeight := [...]int{6: 2, 7: 7, 8: 8, 9: 5, 10: 4, 11: 6, 12: 9, 13: 5,
		14: 4, 15: 4, 16: 6, 17: 10, 18: 11, 19: 8, 20: 6, 21: 4}

	popTotal := 0
	for _, p := range catalog {
		popTotal += p.pop
	}

	var sales []genBill
	// days-1 … 0, so index 0 is the oldest day and the last is today.
	for d := days - 1; d >= 0; d-- {
		day := now.AddDate(0, 0, -d)
		count := 34 + rng.Intn(16) // 34…49 bills on a weekday
		switch day.Weekday() {
		case time.Saturday, time.Sunday:
			count += 14
		case time.Friday:
			count += 6
		}
		// A gentle upward trend, so the 30-day chart tells a story rather than
		// jittering around a flat line.
		count += (days - d) / 6

		for i := 0; i < count; i++ {
			hour := weightedHour(rng, hourWeight[:])
			at := time.Date(day.Year(), day.Month(), day.Day(), hour,
				rng.Intn(60), rng.Intn(60), 0, bangkok)
			// Today is only half-written: no sales from the future.
			if at.After(now) {
				continue
			}

			b := genBill{id: uuid.New(), at: at, method: "cash", cashier: cashierAt(rng, hour)}
			for _, idx := range pickProducts(rng, basketSize(rng), popTotal) {
				b.lines = append(b.lines, genLine{product: idx, qty: pickQty(rng)})
			}
			if rng.Intn(100) < 28 {
				b.method = "transfer"
			}

			var subtotal int64
			for _, ln := range b.lines {
				subtotal += catalog[ln.product].price * int64(ln.qty)
			}
			// Occasional round-number discount, never more than the bill.
			if rng.Intn(100) < 7 {
				d := int64((rng.Intn(4) + 1) * 500) // 5–20 baht
				if d < subtotal {
					b.discount = d
				}
			}
			total := subtotal - b.discount
			b.paid = total
			if b.method == "cash" && rng.Intn(100) < 60 {
				b.paid = roundUpTo(total, 2000) // handed over a 20-baht note or bigger
			}
			sales = append(sales, b)
		}
	}

	// Two or three voids, always on older bills — a void on today's takings
	// would make the "sales today" figure look wrong at a glance.
	var voids []genVoid
	reasons := []string{"ลูกค้าเปลี่ยนใจ", "แคชเชียร์กดสินค้าผิด", "สแกนซ้ำสองครั้ง"}
	older := len(sales) - 60
	for i := 0; i < 3 && older > 10; i++ {
		of := rng.Intn(older)
		voids = append(voids, genVoid{
			id:     uuid.New(),
			at:     sales[of].at.Add(time.Duration(3+rng.Intn(20)) * time.Minute),
			of:     of,
			reason: reasons[i%len(reasons)],
		})
	}
	return sales, voids
}

func weightedHour(rng *rand.Rand, weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	n := rng.Intn(total)
	for hour, w := range weights {
		if n < w {
			return hour
		}
		n -= w
	}
	return 12
}

// basketSize is skewed low: a convenience store sells one or two things at a
// time far more often than a full basket.
func basketSize(rng *rand.Rand) int {
	switch n := rng.Intn(100); {
	case n < 34:
		return 1
	case n < 66:
		return 2
	case n < 85:
		return 3
	case n < 95:
		return 4
	default:
		return 5
	}
}

func pickQty(rng *rand.Rand) int32 {
	switch n := rng.Intn(100); {
	case n < 74:
		return 1
	case n < 92:
		return 2
	default:
		return 3
	}
}

// pickProducts draws `n` distinct products weighted by popularity.
func pickProducts(rng *rand.Rand, n, popTotal int) []int {
	chosen := make([]int, 0, n)
	for len(chosen) < n {
		target := rng.Intn(popTotal)
		idx := 0
		for i, p := range catalog {
			if target < p.pop {
				idx = i
				break
			}
			target -= p.pop
		}
		if !contains(chosen, idx) {
			chosen = append(chosen, idx)
		} else if len(chosen) > 0 && rng.Intn(4) == 0 {
			break // avoid pathological retries on a small catalogue
		}
	}
	return chosen
}

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// cashierAt puts the owner on the early shift and the cashier on the busy
// evening, so the per-user audit trail is not uniform noise.
func cashierAt(rng *rand.Rand, hour int) int {
	switch {
	case hour < 11:
		return []int{0, 0, 1}[rng.Intn(3)]
	case hour < 16:
		return []int{1, 1, 2}[rng.Intn(3)]
	default:
		return []int{2, 2, 2, 1}[rng.Intn(4)]
	}
}

func roundUpTo(v, step int64) int64 {
	if v%step == 0 {
		return v
	}
	return ((v / step) + 1) * step
}
