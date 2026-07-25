# Sabai POS — Frontend (offline-first PWA)

Vite + React + TypeScript · React Router · Dexie/IndexedDB · Zustand · TanStack Query · PWA

ผลลัพธ์ของ `npm run build` ไม่ได้ถูก deploy แยก แต่ถูกฝังเข้าไปใน Go binary
ตอน `docker build` (ดู `Dockerfile` stage แรก) หน้าเว็บกับ API จึงอยู่ origin
เดียวกันเสมอ ทั้งตอน dev และ production

## รัน (dev)

ต้องให้ backend รันอยู่ก่อน (`docker compose up -d` ที่ root → `:8082`)

```bash
npm install
npm run dev          # http://localhost:5173  (proxy /api → :8082)
npm run build        # tsc -b + vite build + service worker
```

> dev server proxy `/api` ไป backend เพื่อให้เป็น same-origin เหมือน production
> (refresh cookie ทำงาน ไม่มี CORS) เปลี่ยนปลายทางด้วย `VITE_API_TARGET`

## สถาปัตยกรรม offline-first

```
src/
  lib/
    api.ts       typed fetch client + auto-refresh (401 → refresh → retry ครั้งเดียว)
    auth.tsx     AuthProvider + RequireAuth (กู้ session จาก refresh cookie)
    meta.ts      ถาม server ว่า deployment นี้เป็นเดโมไหม (bundle เดียวใช้ได้ทั้งสองแบบ)
    db.ts        Dexie schema: products · pending (คิว sync) · bills · meta
    sync.ts      pullCatalog (server→Dexie) · enqueueCheckout (local-first) · pushPending
    promptpay.ts สร้าง payload QR พร้อมเพย์ตามมาตรฐาน EMVCo ในเครื่อง (ออฟไลน์ได้)
    format.ts    baht() แสดงผลจากสตางค์
  store/cart.ts  Zustand cart
  pages/         Login · Cashier · Storefront · Inventory · Dashboard
```

**หลักการ** — แคชเชียร์อ่านสินค้าจาก **IndexedDB** จึงใช้ได้แม้ออฟไลน์ ·
ปิดการขายเขียนลง local แล้วเข้าคิว pending พร้อม **`client_uuid`** ·
sync engine ดันขึ้น server เมื่อออนไลน์ (server เป็น idempotent ตาม `client_uuid`
จึงไม่มีบิลซ้ำแม้ยิงซ้ำ) · ตัดสต็อก local แบบ optimistic แล้ว reconcile ด้วย pullCatalog

**เงินเป็นสตางค์ (integer) ตลอดสาย** ตั้งแต่ API ถึง component — `baht()` แปลง
เป็นข้อความตอนแสดงผลเท่านั้น ไม่มี float โผล่ระหว่างทาง

## หน้าจอ

| เส้นทาง | ใคร | ทำอะไร |
|---|---|---|
| `/login` | ทุกคน | ในโหมดเดโมจะกลายเป็นหน้าแนะนำระบบ + ปุ่มเข้าใช้งานตามบทบาท |
| `/` | ทุกบทบาท | แคชเชียร์ — ยิงบาร์โค้ด, เงินสด/โอน (QR พร้อมเพย์), พิมพ์ใบเสร็จ, จอลูกค้า |
| `/store` | ทุกบทบาท | หน้าร้านสำหรับให้ลูกค้าดู |
| `/inventory` | ดูได้ทุกคน · แก้ได้ manager+ | เพิ่ม/แก้/ลบสินค้า และรับสต็อกเข้า |
| `/dashboard` | manager+ | ยอดขาย กำไร สินค้าขายดี สินค้าใกล้หมด |

สิทธิ์ถูกบังคับที่เซิร์ฟเวอร์ ไม่ใช่แค่ซ่อนปุ่ม — `cashier` ที่ยิง
`/api/v1/reports/summary` ตรง ๆ จะได้ 403

## ที่ยังไม่ได้ทำ

การ "นำเข้าสินค้า" แบบ 3 แท็บ (สแกน/CSV/PO) ในดีไซน์ต้นฉบับ ยังไม่ทำในรอบนี้
ตอนนี้ใช้เพิ่ม/รับเข้าทีละรายการแทน
