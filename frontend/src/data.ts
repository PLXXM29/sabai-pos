// ─────────────────────────────────────────────────────────────────────────
// MiniMart POS — seed data & domain types
// (ported verbatim from the MiniMart POS design)
// ─────────────────────────────────────────────────────────────────────────

export type Product = {
  id: number
  name: string
  cat: string
  cost: number
  price: number
  stock: number
  sold: number
  barcode: string
}

export type CartLine = {
  id: number
  name: string
  price: number
  cost: number
  cat: string
  qty: number
}

export type Tx = { id: number; total: number; profit: number; items: number }

export type Receipt = {
  bill: number
  rows: { name: string; qty: number; lineText: string }[]
  total: number
  received: number
  change: number
  method: 'cash' | 'transfer'
  time: string
}

export type SupplierItem = {
  id?: number
  name?: string
  cat?: string
  cost?: number
  price?: number
  qty: number
}
export type Supplier = { id: string; name: string; po: string; items: SupplierItem[] }

export const CATS = ['เครื่องดื่ม', 'ขนม', 'อาหาร', 'ของใช้', 'บุหรี่/อื่นๆ']

/** [tile background, tile foreground] per category */
export const TILES: Record<string, [string, string]> = {
  เครื่องดื่ม: ['#DCEDEA', '#17706A'],
  ขนม: ['#F6E3C2', '#A66A10'],
  อาหาร: ['#F3DACB', '#B0511D'],
  ของใช้: ['#E7E8D6', '#6B7040'],
  'บุหรี่/อื่นๆ': ['#E5E0D8', '#6E6355'],
}

const P = (
  id: number,
  name: string,
  cat: string,
  cost: number,
  price: number,
  stock: number,
  sold: number,
): Product => ({
  id,
  name,
  cat,
  cost,
  price,
  stock,
  sold,
  barcode: '885' + String(1000000000 + id * 7919).slice(0, 10),
})

export const SEED_PRODUCTS: Product[] = [
  P(1, 'น้ำดื่ม 600 มล.', 'เครื่องดื่ม', 4, 7, 48, 96),
  P(2, 'น้ำอัดลมโคล่า 325 มล.', 'เครื่องดื่ม', 10, 15, 24, 71),
  P(3, 'ชาเขียวพร้อมดื่ม', 'เครื่องดื่ม', 14, 20, 18, 42),
  P(4, 'กาแฟกระป๋อง', 'เครื่องดื่ม', 10, 15, 30, 55),
  P(5, 'นมเปรี้ยวขวดเล็ก', 'เครื่องดื่ม', 8, 12, 7, 38),
  P(6, 'เครื่องดื่มชูกำลัง', 'เครื่องดื่ม', 7, 10, 36, 64),
  P(7, 'น้ำผลไม้กล่อง', 'เครื่องดื่ม', 18, 25, 12, 19),
  P(8, 'มันฝรั่งทอดกรอบ', 'ขนม', 14, 20, 22, 58),
  P(9, 'เวเฟอร์ช็อกโกแลต', 'ขนม', 6, 10, 40, 47),
  P(10, 'ปลาเส้นรสเผ็ด', 'ขนม', 8, 12, 15, 33),
  P(11, 'สาหร่ายทอดกรอบ', 'ขนม', 14, 20, 9, 41),
  P(12, 'ลูกอมมินต์', 'ขนม', 3, 5, 60, 28),
  P(13, 'บิสกิตแซนวิชครีม', 'ขนม', 8, 12, 26, 22),
  P(14, 'ขนมปังไส้สังขยา', 'ขนม', 12, 18, 6, 36),
  P(15, 'บะหมี่กึ่งสำเร็จรูป', 'อาหาร', 4.5, 6, 72, 88),
  P(16, 'ข้าวกล่องแช่เย็น', 'อาหาร', 28, 39, 8, 26),
  P(17, 'ไส้กรอกชีส', 'อาหาร', 10, 15, 20, 52),
  P(18, 'ซาลาเปาไส้หมู', 'อาหาร', 15, 22, 10, 31),
  P(19, 'ไข่ต้ม 2 ฟอง', 'อาหาร', 10, 15, 14, 24),
  P(20, 'แซนด์วิชแฮมชีส', 'อาหาร', 22, 32, 5, 18),
  P(21, 'ยาสีฟัน 90 ก.', 'ของใช้', 26, 35, 12, 9),
  P(22, 'แปรงสีฟัน', 'ของใช้', 17, 25, 16, 7),
  P(23, 'สบู่ก้อน', 'ของใช้', 12, 18, 20, 11),
  P(24, 'แชมพูซอง', 'ของใช้', 3.5, 6, 80, 25),
  P(25, 'ผงซักฟอก 400 ก.', 'ของใช้', 24, 32, 9, 8),
  P(26, 'กระดาษทิชชู่ม้วน', 'ของใช้', 20, 29, 14, 13),
  P(27, 'ถ่านไฟฉาย AA (คู่)', 'ของใช้', 32, 45, 0, 5),
  P(28, 'บุหรี่ (ซอง)', 'บุหรี่/อื่นๆ', 65, 72, 25, 46),
  P(29, 'ไฟแช็กแก๊ส', 'บุหรี่/อื่นๆ', 6, 10, 33, 21),
  P(30, 'ถุงขยะ 10 ใบ', 'บุหรี่/อื่นๆ', 15, 22, 11, 6),
]

export const SEED_TX: Omit<Tx, 'id'>[] = [
  { total: 87, profit: 24, items: 5 },
  { total: 156, profit: 41, items: 8 },
  { total: 45, profit: 12, items: 3 },
  { total: 210, profit: 55, items: 9 },
  { total: 32, profit: 9, items: 2 },
  { total: 128, profit: 33, items: 6 },
  { total: 74, profit: 20, items: 4 },
  { total: 189, profit: 48, items: 7 },
  { total: 96, profit: 26, items: 5 },
  { total: 61, profit: 17, items: 3 },
  { total: 143, profit: 37, items: 6 },
  { total: 258, profit: 66, items: 11 },
  { total: 52, profit: 14, items: 3 },
  { total: 118, profit: 30, items: 5 },
  { total: 83, profit: 22, items: 4 },
]

export const SUPPLIERS: Supplier[] = [
  {
    id: 'cpall',
    name: 'ซีพี ออลล์ · เครื่องดื่ม/ขนม',
    po: 'PO-2569-0712',
    items: [
      { id: 1, qty: 48 },
      { id: 2, qty: 24 },
      { id: 6, qty: 36 },
      { id: 8, qty: 36 },
      { name: 'โซดาขวดเล็ก', cat: 'เครื่องดื่ม', cost: 8, price: 12, qty: 24 },
    ],
  },
  {
    id: 'sahapat',
    name: 'สหพัฒน์ · ของใช้/อาหาร',
    po: 'PO-2569-0713',
    items: [
      { id: 15, qty: 120 },
      { id: 24, qty: 100 },
      { id: 26, qty: 30 },
      { name: 'ผ้าเปียกเช็ดทำความสะอาด', cat: 'ของใช้', cost: 18, price: 29, qty: 20 },
      { name: 'โจ๊กคัพกึ่งสำเร็จรูป', cat: 'อาหาร', cost: 12, price: 18, qty: 36 },
    ],
  },
  {
    id: 'coke',
    name: 'ไทยน้ำทิพย์ · น้ำอัดลม',
    po: 'PO-2569-0715',
    items: [
      { id: 2, qty: 48 },
      { id: 4, qty: 24 },
      { name: 'น้ำอัดลมส้ม 325 มล.', cat: 'เครื่องดื่ม', cost: 10, price: 15, qty: 48 },
    ],
  },
]

export const WEEK = [
  { label: 'ส.', val: 8420 },
  { label: 'อา.', val: 9150 },
  { label: 'จ.', val: 6890 },
  { label: 'อ.', val: 7340 },
  { label: 'พ.', val: 7020 },
  { label: 'พฤ.', val: 7960 },
]
