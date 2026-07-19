# ADR 0001 — เก็บเงินเป็น integer satang

- สถานะ: Accepted
- วันที่: 2026-07-19

## Context
ระบบ POS จัดการเงินจริง การใช้ `float`/`double` ทำให้เกิด rounding error
(เช่น 0.1 + 0.2 ≠ 0.3) ซึ่งยอมรับไม่ได้กับยอดขาย/เงินทอน/กำไร
ต้องเลือกวิธีแทนค่าเงินให้ **แม่นยำแบบ exact** และใช้เหมือนกันทั้งระบบ
(Postgres, Go, และ frontend)

ทางเลือกที่พิจารณา:
1. **Integer satang** — เก็บเป็นจำนวนเต็มหน่วยสตางค์ (1 บาท = 100 สตางค์)
2. `NUMERIC(12,2)` ใน Postgres + `shopspring/decimal` ใน Go

## Decision
เลือก **Integer satang** — เจ้าของโปรเจกต์ยืนยัน

- Postgres: คอลัมน์เงินทุกตัวเป็น `BIGINT` หน่วยสตางค์ (`sell_price`, `cost_price`,
  `subtotal`, `discount`, `total`, `paid`, `change`, `price_snapshot`, `line_total`)
- Go: ใช้ `int64` ตลอด ไม่มี float ในเส้นทางเงิน
- Frontend: คำนวณด้วย integer สตางค์ แปลงเป็นบาทเฉพาะตอนแสดงผล (`฿` + `satang/100`)
- กติกาปัดเศษ: การคำนวณที่อาจได้เศษ (เช่น ภาษี/ส่วนลด %) ให้ปัดเป็นสตางค์
  ด้วยกฎเดียวกันทั้งระบบ (round half-up ที่ระดับ service)

## Consequences
- ✅ Exact ไม่มี floating error, เปรียบเทียบ/รวมยอดเชื่อถือได้
- ✅ ไม่ต้องพึ่ง decimal library, เร็ว, serialize เป็น JSON number ได้ตรงๆ
- ✅ Index/aggregate บน BIGINT เร็ว
- ⚠️ ต้องระวัง "อย่าเผลอหาร/คูณแล้วเก็บกลับเป็น float" — บังคับผ่าน type `int64`
  และ helper แปลงหน่วยที่จุดแสดงผลเท่านั้น
- ⚠️ ถ้าอนาคตต้องรองรับหลายสกุลเงิน/ทศนิยม >2 ตำแหน่ง ต้องทบทวน (ตอนนี้ THB พอ)
