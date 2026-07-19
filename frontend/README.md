# MiniMart POS — Frontend (offline-first PWA)

Vite + React + TypeScript · React Router · Dexie/IndexedDB · Zustand · TanStack Query · PWA

## รัน (dev)

ต้องให้ backend รันอยู่ก่อน (`docker compose up -d` ที่ root → API ที่ `:8082`)

```bash
npm install
npm run dev          # http://localhost:5173  (proxy /api → :8082)
npm run build        # + service worker (PWA)
```

> dev server proxy `/api` → backend ทำให้เป็น same-origin (cookie refresh ทำงาน, ไม่มี CORS)
> เปลี่ยนปลายทางได้ด้วย env `VITE_API_TARGET`

## สถาปัตยกรรม offline-first

```
src/
  lib/
    api.ts     typed fetch client + auto-refresh (401 → refresh → retry)
    auth.tsx   AuthProvider + RequireAuth (กู้ session จาก refresh cookie)
    db.ts      Dexie schema: products · pending (คิว sync) · bills · meta
    sync.ts    pullCatalog (server→Dexie) · enqueueCheckout (local-first) · pushPending
    format.ts  baht() แสดงผลจาก satang
  store/cart.ts  Zustand cart
  pages/         Login · Cashier (+ Storefront/Inventory/Dashboard = Phase 3b)
```

**หลักการ:** แคชเชียร์อ่านสินค้าจาก **IndexedDB** (ใช้ได้แม้ออฟไลน์) · ปิดการขายเขียนลง local +
เข้าคิว pending พร้อม **`client_uuid`** · sync engine ดันขึ้น server เมื่อออนไลน์ (idempotent
ฝั่ง server กันบิลซ้ำ) · ตัดสต็อก local แบบ optimistic แล้ว reconcile ด้วย pullCatalog

## สถานะ (Phase 3 เสร็จ)

✅ Login/auth (JWT + refresh cookie) · ✅ Cashier offline-first (ขายเงินสด/โอน → sync → ใบเสร็จ) ·
✅ หน้าร้าน · ✅ จัดการสต็อก (เพิ่ม/แก้/ลบ/รับเข้า, RBAC — cashier อ่านอย่างเดียว) ·
✅ แดชบอร์ด (React Query → report endpoints) · ✅ จอลูกค้า (sync กับ cart) ·
✅ ยิงบาร์โค้ด (keyboard-wedge ที่ช่องค้นหา) · ✅ PWA installable

หมายเหตุ: การ"นำเข้าสินค้า" แบบ 3 แท็บ (สแกน/CSV/PO) ในดีไซน์เดิม ยังไม่ทำในรอบนี้
(ใช้เพิ่ม/รับเข้าทีละรายการแทน) — เพิ่มภายหลังได้
