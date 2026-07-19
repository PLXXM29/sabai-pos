# ADR 0002 — โครงสร้างโปรเจกต์และ stack

- สถานะ: Accepted
- วันที่: 2026-07-19

## Context
ต้องยกระดับ prototype (Vite/React ตัวเดียว) เป็นระบบ production ที่ offline-first,
multi-tenant-ready, maintain ง่าย ตาม build brief ที่เจ้าของกำหนด

## Decision
- **Monorepo 2 ส่วน**: `backend/` (Go API) + `frontend/` (Vite POS client) + `docs/`
- **Backend**: Go 1.25+/Gin, PostgreSQL 16, **sqlc** (type-safe queries — ไม่ใช้ ORM
  กับตารางเงิน), golang-migrate, JWT auth, Zap logging
- **Layered architecture**: `handler` (บาง) → `service` (business logic) →
  `store` (sqlc) ; `domain` ไม่พึ่ง framework ; DB access ผ่าน interface (mock ได้)
- **Frontend**: Vite + React 18 + TS strict, React Router, Dexie/IndexedDB (local-first),
  vite-plugin-pwa, TanStack Query, Zustand, Tailwind (Phase 3)
- **Multi-tenant-ready**: ทุกตารางมี `store_id` ตั้งแต่วันแรก แต่ deploy ร้านเดียวก่อน
  ยังไม่ทำ billing/subscription
- **Money**: integer satang (ดู [ADR 0001](0001-money-representation.md))

## หลักการฐานข้อมูลที่ยึด
- `stock_movements` = **immutable ledger** (append-only, กัน UPDATE/DELETE ด้วย trigger)
  คำนวณ qty คงเหลือจาก ledger ได้เสมอ; `inventory` เป็น cache ที่อัปเดตใน tx เดียวกัน
- `bills` = เอกสารการเงิน **immutable**; ยกเลิก = สร้าง void/refund record ใหม่ที่อ้างบิลเดิม
  (`voids_bill_id`) ไม่ลบของเดิม
- `bill_items` เก็บ **snapshot** ชื่อ+ราคา ณ ตอนขาย — แก้ราคาสินค้าทีหลังไม่กระทบบิลเก่า
- `client_uuid` เป็น **idempotency key** (unique ต่อร้าน) กันการ sync ซ้ำสร้างบิลซ้ำ
- `bill_counters` ออกเลขบิล **gap-free ต่อร้าน** ฝั่ง server ตอน commit/sync

## Consequences
- ✅ ตรวจสอบย้อนหลังได้ (audit), ยอดเงิน/สต็อกพิสูจน์จาก ledger ได้
- ✅ รองรับ offline sync โดยไม่ lost-update (สต็อกรวมจาก movement ไม่ overwrite)
- ⚠️ ต้องเขียน service layer ให้ทำงานใน DB transaction จริงเสมอ (atomic: บิล+items+สต็อก)
- ⚠️ sqlc ต้อง regenerate ทุกครั้งที่แก้ schema/queries (`make sqlc`)
