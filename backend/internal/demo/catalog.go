package demo

// The sample store. Names are generic Thai convenience-store lines (no real
// brands), prices are what a Bangkok shop actually charges, and every amount is
// satang — 7.00 baht is 700, never 7.0.
//
// `onHand` is the stock the demo should *end* on, not the amount received:
// rebuild() receives however much the generated sales history consumed plus this
// number, so the ledger, the inventory cache and the sales history all agree.
// A few lines sit at or under their reorder point on purpose, so the dashboard's
// low-stock warning and the storefront's out-of-stock state have something real
// to show.

const shopName = "มาร์ทมงคลชัย"

type product struct {
	id       int
	name     string
	category string
	cost     int64 // satang
	price    int64 // satang
	onHand   int32
	reorder  int32
	pop      int // relative basket popularity, 1 (rare) … 10 (every other bill)
}

var catalog = []product{
	{1, "น้ำดื่ม 600 มล.", "เครื่องดื่ม", 400, 700, 48, 24, 10},
	{2, "น้ำอัดลมโคล่า 325 มล.", "เครื่องดื่ม", 1000, 1500, 24, 12, 8},
	{3, "ชาเขียวพร้อมดื่ม", "เครื่องดื่ม", 1400, 2000, 18, 12, 7},
	{4, "กาแฟกระป๋อง", "เครื่องดื่ม", 1000, 1500, 30, 12, 8},
	{5, "นมเปรี้ยวขวดเล็ก", "เครื่องดื่ม", 800, 1200, 7, 10, 6},
	{6, "เครื่องดื่มชูกำลัง", "เครื่องดื่ม", 700, 1000, 36, 18, 9},
	{7, "น้ำผลไม้กล่อง", "เครื่องดื่ม", 1800, 2500, 12, 8, 4},
	{8, "นมกล่อง UHT 200 มล.", "เครื่องดื่ม", 900, 1300, 26, 12, 6},
	{9, "มันฝรั่งทอดกรอบ", "ขนม", 1400, 2000, 22, 10, 7},
	{10, "เวเฟอร์ช็อกโกแลต", "ขนม", 600, 1000, 40, 15, 6},
	{11, "ปลาเส้นรสเผ็ด", "ขนม", 800, 1200, 15, 10, 5},
	{12, "สาหร่ายทอดกรอบ", "ขนม", 1400, 2000, 9, 10, 4},
	{13, "ลูกอมมินต์", "ขนม", 300, 500, 60, 20, 5},
	{14, "บิสกิตแซนวิชครีม", "ขนม", 800, 1200, 26, 12, 5},
	{15, "ขนมปังไส้สังขยา", "ขนม", 1200, 1800, 6, 8, 6},
	{16, "ถั่วอบเกลือ ซองเล็ก", "ขนม", 700, 1100, 18, 10, 3},
	{17, "บะหมี่กึ่งสำเร็จรูป", "อาหาร", 450, 600, 72, 30, 10},
	{18, "ข้าวกล่องแช่เย็น", "อาหาร", 2800, 3900, 8, 6, 5},
	{19, "ไส้กรอกชีส", "อาหาร", 1000, 1500, 20, 10, 6},
	{20, "ซาลาเปาไส้หมู", "อาหาร", 1500, 2200, 10, 8, 5},
	{21, "ไข่ต้ม 2 ฟอง", "อาหาร", 1000, 1500, 14, 10, 5},
	{22, "แซนด์วิชแฮมชีส", "อาหาร", 2200, 3200, 5, 6, 4},
	{23, "ไข่ไก่ เบอร์ 2 (แผง 10)", "อาหาร", 4200, 5500, 12, 6, 4},
	{24, "ยาสีฟัน 90 ก.", "ของใช้", 2600, 3500, 12, 6, 2},
	{25, "แปรงสีฟัน", "ของใช้", 1700, 2500, 16, 8, 2},
	{26, "สบู่ก้อน", "ของใช้", 1200, 1800, 20, 10, 3},
	{27, "แชมพูซอง", "ของใช้", 350, 600, 80, 30, 4},
	{28, "ผงซักฟอก 400 ก.", "ของใช้", 2400, 3200, 9, 8, 2},
	{29, "กระดาษทิชชู่ม้วน", "ของใช้", 2000, 2900, 14, 8, 3},
	{30, "ถุงขยะ 10 ใบ", "ของใช้", 1500, 2200, 11, 8, 2},
	{31, "ถ่านไฟฉาย AA (คู่)", "ของใช้", 3200, 4500, 0, 4, 1},
	{32, "ยาแก้ปวดลดไข้ แผง", "ของใช้", 900, 1500, 24, 10, 3},
	{33, "บุหรี่ (ซอง)", "บุหรี่/อื่นๆ", 6500, 7200, 25, 10, 7},
	{34, "ไฟแช็กแก๊ส", "บุหรี่/อื่นๆ", 600, 1000, 33, 12, 4},
	{35, "หนังยาง/ถุงหูหิ้ว", "บุหรี่/อื่นๆ", 900, 1400, 16, 8, 2},
	{36, "บัตรเติมเงินมือถือ 50", "บุหรี่/อื่นๆ", 4700, 5000, 20, 10, 5},
}

// Account is a demo login. In demo mode these are published by /api/v1/meta so
// the sign-in screen can offer one-click entry — see meta_handler.go for why
// that is safe here and nowhere else.
type Account struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Accounts covers the three permission levels the RBAC layer distinguishes, so a
// visitor can see for themselves that a cashier really is refused the manager's
// screens rather than just having them hidden.
var Accounts = []Account{
	{
		Username:    "owner",
		Password:    "owner1234",
		Role:        "superadmin",
		Label:       "เจ้าของร้าน",
		Description: "เห็นทุกอย่าง — ยอดขาย กำไร ต้นทุน จัดการสินค้าและพนักงาน",
	},
	{
		Username:    "manager",
		Password:    "manager1234",
		Role:        "manager",
		Label:       "ผู้จัดการ",
		Description: "ขายได้ รับสต็อก แก้ราคา ยกเลิกบิล ดูรายงาน",
	},
	{
		Username:    "cashier",
		Password:    "cashier1234",
		Role:        "cashier",
		Label:       "พนักงานแคชเชียร์",
		Description: "ขายและพิมพ์ใบเสร็จเท่านั้น — รายงานกำไรและการแก้ราคาถูกปิด",
	},
}
