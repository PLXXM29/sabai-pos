//go:build integration

// Integration tests run against a real PostgreSQL started via testcontainers.
//
//	go test -tags=integration ./internal/store/...
package store_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"sabai-pos/backend/internal/config"
	"sabai-pos/backend/internal/domain"
	"sabai-pos/backend/internal/service"
	"sabai-pos/backend/internal/store"
)

var (
	testDB   *store.DB
	testPool *pgxpool.Pool
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90*time.Second)),
	)
	if err != nil {
		log.Fatalf("start postgres: %v", err)
	}
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("dsn: %v", err)
	}
	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}
	if err := applyMigrations(ctx, testPool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	testDB = store.NewDB(testPool)

	code := m.Run()

	testPool.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}

// freshStore + a cashier user, isolating each test's data.
func freshStore(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	st, err := testDB.CreateStore(ctx, store.CreateStoreParams{Name: "t-" + uuid.NewString(), Config: []byte("{}")})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	u, err := testDB.CreateUser(ctx, store.CreateUserParams{
		StoreID: st.ID, Username: "cash-" + uuid.NewString(), PasswordHash: "x", Role: "cashier",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return st.ID, u.ID
}

func newProduct(t *testing.T, storeID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	p, err := testDB.CreateProduct(ctx, store.CreateProductParams{
		StoreID: storeID, Name: "น้ำดื่ม", Category: "เครื่องดื่ม", CostPrice: 400, SellPrice: 700,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if _, err := testDB.CreateInventory(ctx, store.CreateInventoryParams{
		ProductID: p.ID, StoreID: storeID, QtyOnHand: 0, ReorderPoint: 10,
	}); err != nil {
		t.Fatalf("create inventory: %v", err)
	}
	return p.ID
}

func TestStockReceiveIsIdempotentAndLedgerMatches(t *testing.T) {
	ctx := context.Background()
	storeID, userID := freshStore(t)
	productID := newProduct(t, storeID)
	stock := service.NewStockService(testDB)

	cu := uuid.New()
	inv, err := stock.Receive(ctx, storeID, productID, 48, "first", userID, &cu)
	if err != nil {
		t.Fatalf("receive #1: %v", err)
	}
	if inv.QtyOnHand != 48 {
		t.Fatalf("qty after receive #1 = %d, want 48", inv.QtyOnHand)
	}

	// Same client_uuid → must NOT double-count (offline sync retry).
	inv2, err := stock.Receive(ctx, storeID, productID, 48, "dup", userID, &cu)
	if err != nil {
		t.Fatalf("receive #2: %v", err)
	}
	if inv2.QtyOnHand != 48 {
		t.Fatalf("idempotency broken: qty = %d, want 48", inv2.QtyOnHand)
	}

	// A different receive adds.
	cu2 := uuid.New()
	inv3, err := stock.Receive(ctx, storeID, productID, 10, "second", userID, &cu2)
	if err != nil {
		t.Fatalf("receive #3: %v", err)
	}
	if inv3.QtyOnHand != 58 {
		t.Fatalf("qty = %d, want 58", inv3.QtyOnHand)
	}

	// Cached on-hand must equal the ledger sum (source of truth).
	onHand, err := stock.OnHand(ctx, productID)
	if err != nil {
		t.Fatalf("onhand: %v", err)
	}
	if onHand != 58 {
		t.Fatalf("ledger on-hand = %d, want 58", onHand)
	}
}

func TestStockMovementsAreAppendOnly(t *testing.T) {
	ctx := context.Background()
	storeID, userID := freshStore(t)
	productID := newProduct(t, storeID)
	if _, err := testDB.CreateMovement(ctx, store.CreateMovementParams{
		StoreID: storeID, ProductID: productID, Type: "receive", QtyDelta: 5, CreatedBy: store.PgUUID(userID),
	}); err != nil {
		t.Fatalf("create movement: %v", err)
	}
	// The ledger must reject mutation (enforced by DB trigger).
	if _, err := testPool.Exec(ctx, "UPDATE stock_movements SET qty_delta = 999 WHERE product_id = $1", productID); err == nil {
		t.Fatal("expected append-only ledger to reject UPDATE")
	}
	if _, err := testPool.Exec(ctx, "DELETE FROM stock_movements WHERE product_id = $1", productID); err == nil {
		t.Fatal("expected append-only ledger to reject DELETE")
	}
}

func TestCheckoutDeductsStockIdempotentlyAndVoidRestores(t *testing.T) {
	ctx := context.Background()
	storeID, userID := freshStore(t)
	productID := newProduct(t, storeID) // sell_price 700 satang
	stock := service.NewStockService(testDB)
	sale := service.NewSaleService(testDB)

	// Stock the shelf: 50 units.
	if _, err := stock.Receive(ctx, storeID, productID, 50, "init", userID, nil); err != nil {
		t.Fatalf("receive: %v", err)
	}

	// Checkout 3 units, cash, paid 2500 → total 2100, change 400.
	cu := uuid.New()
	in := service.CheckoutInput{
		Lines:         []service.CheckoutLine{{ProductID: productID, Qty: 3}},
		PaymentMethod: "cash", Paid: 2500, ClientUUID: &cu,
	}
	d, err := sale.Checkout(ctx, storeID, in, userID)
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if d.Bill.Total != 2100 || d.Bill.Change != 400 {
		t.Fatalf("bill total=%d change=%d, want 2100/400", d.Bill.Total, d.Bill.Change)
	}
	if on, _ := stock.OnHand(ctx, productID); on != 47 {
		t.Fatalf("on-hand after sale = %d, want 47", on)
	}

	// Same client_uuid → idempotent (no second deduction).
	d2, err := sale.Checkout(ctx, storeID, in, userID)
	if err != nil {
		t.Fatalf("checkout retry: %v", err)
	}
	if d2.Bill.ID != d.Bill.ID {
		t.Fatal("idempotency broken: retry created a new bill")
	}
	if on, _ := stock.OnHand(ctx, productID); on != 47 {
		t.Fatalf("on-hand after retry = %d, want 47 (no double deduction)", on)
	}

	// Oversell must be rejected.
	big := uuid.New()
	if _, err := sale.Checkout(ctx, storeID, service.CheckoutInput{
		Lines: []service.CheckoutLine{{ProductID: productID, Qty: 9999}}, PaymentMethod: "cash", Paid: 9_999_999, ClientUUID: &big,
	}, userID); err == nil {
		t.Fatal("expected oversell to be rejected")
	}

	// Void restores stock and cannot be repeated.
	if _, err := sale.Void(ctx, storeID, d.Bill.ID, "test refund", userID); err != nil {
		t.Fatalf("void: %v", err)
	}
	if on, _ := stock.OnHand(ctx, productID); on != 50 {
		t.Fatalf("on-hand after void = %d, want 50 (restored)", on)
	}
	if _, err := sale.Void(ctx, storeID, d.Bill.ID, "again", userID); err == nil {
		t.Fatal("expected double-void to be rejected")
	}
}

func TestAuthRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	storeID, _ := freshStore(t)
	cfg := &config.Config{
		AppEnv:           "development",
		JWTAccessSecret:  "test-access-secret-0123456789abcdef",
		JWTRefreshSecret: "test-refresh-secret-0123456789abcdef",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  24 * time.Hour,
	}
	as := service.NewAuthService(testDB, cfg)

	if _, err := as.Register(ctx, storeID, "boss", "pw123456", domain.RoleManager); err != nil {
		t.Fatalf("register: %v", err)
	}
	pair, err := as.Login(ctx, "boss", "pw123456")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}
	if _, err := as.Login(ctx, "boss", "wrong-pw"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("wrong password: want ErrInvalidCredentials, got %v", err)
	}

	// Change password: wrong current rejected; correct current rotates it.
	if err := as.ChangePassword(ctx, pair.User.ID, "nope", "brandnew1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("change with wrong current: want ErrInvalidCredentials, got %v", err)
	}
	if err := as.ChangePassword(ctx, pair.User.ID, "pw123456", "brandnew1"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, err := as.Login(ctx, "boss", "pw123456"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatal("old password must no longer work")
	}
	if _, err := as.Login(ctx, "boss", "brandnew1"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}
