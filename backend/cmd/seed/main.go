// Command seed populates a fresh database with a default store, staff users and
// the 30 demo products (prices in satang). Idempotent: safe to run repeatedly.
//
//	DATABASE_URL=postgres://... go run ./cmd/seed
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"minimart-pos/backend/internal/auth"
	"minimart-pos/backend/internal/store"
)

const shopName = "มาร์ทมงคลชัย"

type seedUser struct {
	username, password, role string
}

var seedUsers = []seedUser{
	{"owner", "owner1234", "superadmin"},
	{"manager", "manager1234", "manager"},
	{"cashier", "cashier1234", "cashier"},
}

type seedProduct struct {
	id    int
	name  string
	cat   string
	cost  int64 // satang
	price int64 // satang
	stock int32
}

// Prices/costs in satang (baht × 100; 4.5 baht = 450).
var seedProducts = []seedProduct{
	{1, "น้ำดื่ม 600 มล.", "เครื่องดื่ม", 400, 700, 48},
	{2, "น้ำอัดลมโคล่า 325 มล.", "เครื่องดื่ม", 1000, 1500, 24},
	{3, "ชาเขียวพร้อมดื่ม", "เครื่องดื่ม", 1400, 2000, 18},
	{4, "กาแฟกระป๋อง", "เครื่องดื่ม", 1000, 1500, 30},
	{5, "นมเปรี้ยวขวดเล็ก", "เครื่องดื่ม", 800, 1200, 7},
	{6, "เครื่องดื่มชูกำลัง", "เครื่องดื่ม", 700, 1000, 36},
	{7, "น้ำผลไม้กล่อง", "เครื่องดื่ม", 1800, 2500, 12},
	{8, "มันฝรั่งทอดกรอบ", "ขนม", 1400, 2000, 22},
	{9, "เวเฟอร์ช็อกโกแลต", "ขนม", 600, 1000, 40},
	{10, "ปลาเส้นรสเผ็ด", "ขนม", 800, 1200, 15},
	{11, "สาหร่ายทอดกรอบ", "ขนม", 1400, 2000, 9},
	{12, "ลูกอมมินต์", "ขนม", 300, 500, 60},
	{13, "บิสกิตแซนวิชครีม", "ขนม", 800, 1200, 26},
	{14, "ขนมปังไส้สังขยา", "ขนม", 1200, 1800, 6},
	{15, "บะหมี่กึ่งสำเร็จรูป", "อาหาร", 450, 600, 72},
	{16, "ข้าวกล่องแช่เย็น", "อาหาร", 2800, 3900, 8},
	{17, "ไส้กรอกชีส", "อาหาร", 1000, 1500, 20},
	{18, "ซาลาเปาไส้หมู", "อาหาร", 1500, 2200, 10},
	{19, "ไข่ต้ม 2 ฟอง", "อาหาร", 1000, 1500, 14},
	{20, "แซนด์วิชแฮมชีส", "อาหาร", 2200, 3200, 5},
	{21, "ยาสีฟัน 90 ก.", "ของใช้", 2600, 3500, 12},
	{22, "แปรงสีฟัน", "ของใช้", 1700, 2500, 16},
	{23, "สบู่ก้อน", "ของใช้", 1200, 1800, 20},
	{24, "แชมพูซอง", "ของใช้", 350, 600, 80},
	{25, "ผงซักฟอก 400 ก.", "ของใช้", 2400, 3200, 9},
	{26, "กระดาษทิชชู่ม้วน", "ของใช้", 2000, 2900, 14},
	{27, "ถ่านไฟฉาย AA (คู่)", "ของใช้", 3200, 4500, 0},
	{28, "บุหรี่ (ซอง)", "บุหรี่/อื่นๆ", 6500, 7200, 25},
	{29, "ไฟแช็กแก๊ส", "บุหรี่/อื่นๆ", 600, 1000, 33},
	{30, "ถุงขยะ 10 ใบ", "บุหรี่/อื่นๆ", 1500, 2200, 11},
}

func barcodeFor(id int) string { return fmt.Sprintf("885%d", 1000000000+id*7919) }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed error:", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	db := store.NewDB(pool)

	// 1. Store (idempotent by name).
	st, err := findStoreByName(ctx, db, shopName)
	if err != nil {
		return err
	}
	if st == nil {
		created, err := db.CreateStore(ctx, store.CreateStoreParams{Name: shopName, Config: []byte("{}")})
		if err != nil {
			return err
		}
		st = &created
		fmt.Println("created store:", st.ID)
	} else {
		fmt.Println("store exists:", st.ID)
	}

	// 2. Users.
	usersAdded := 0
	for _, su := range seedUsers {
		if _, err := db.GetActiveUserByUsername(ctx, su.username); err == nil {
			continue // already there
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		hash, err := auth.HashPassword(su.password)
		if err != nil {
			return err
		}
		if _, err := db.CreateUser(ctx, store.CreateUserParams{
			StoreID: st.ID, Username: su.username, PasswordHash: hash, Role: su.role,
		}); err != nil {
			return err
		}
		usersAdded++
	}

	// 3. Products + initial stock via ledger (idempotent by barcode).
	prodAdded := 0
	for _, p := range seedProducts {
		bc := barcodeFor(p.id)
		if _, err := db.GetProductByBarcode(ctx, store.GetProductByBarcodeParams{StoreID: st.ID, Barcode: &bc}); err == nil {
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		err := db.ExecTx(ctx, func(q *store.Queries) error {
			prod, err := q.CreateProduct(ctx, store.CreateProductParams{
				StoreID: st.ID, Name: p.name, Barcode: &bc, Category: p.cat,
				CostPrice: p.cost, SellPrice: p.price,
			})
			if err != nil {
				return err
			}
			if _, err := q.CreateInventory(ctx, store.CreateInventoryParams{
				ProductID: prod.ID, StoreID: st.ID, QtyOnHand: 0, ReorderPoint: 10,
			}); err != nil {
				return err
			}
			if p.stock > 0 {
				reason := "seed initial stock"
				if _, err := q.CreateMovement(ctx, store.CreateMovementParams{
					StoreID: st.ID, ProductID: prod.ID, Type: "receive",
					QtyDelta: p.stock, Reason: &reason,
				}); err != nil {
					return err
				}
				if _, err := q.AddInventoryQty(ctx, store.AddInventoryQtyParams{
					ProductID: prod.ID, QtyOnHand: p.stock,
				}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		prodAdded++
	}

	fmt.Printf("seed done · users +%d · products +%d\n", usersAdded, prodAdded)
	fmt.Println("dev logins: owner/owner1234 (superadmin), manager/manager1234, cashier/cashier1234")
	return nil
}

func findStoreByName(ctx context.Context, db *store.DB, name string) (*store.Store, error) {
	stores, err := db.ListStores(ctx)
	if err != nil {
		return nil, err
	}
	for i := range stores {
		if stores[i].Name == name {
			return &stores[i], nil
		}
	}
	return nil, nil
}
