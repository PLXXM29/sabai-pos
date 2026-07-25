# Sabai POS — Backend

Go 1.25 · Gin · PostgreSQL 16 · sqlc · JWT · Zap

ไบนารีตัวนี้ทำสามอย่างในตัวเดียว: เสิร์ฟ REST API, รัน migration ตอนบูต,
และเสิร์ฟหน้าเว็บที่ฝัง (`embed.FS`) ไว้ข้างใน จึงไม่มี reverse proxy หรือ
migrate job แยกให้ดูแล — เหตุผลอยู่ใน [`internal/web/web.go`](internal/web/web.go)

## คำสั่งที่ใช้บ่อย

```bash
make run          # รัน API (ต้องมี DATABASE_URL ที่ต่อได้)
make ui           # build หน้าเว็บเข้าไปไว้ในที่ที่ไบนารีฝัง
make ui-reset     # คืน placeholder หลังใช้ make ui
make test         # unit (ไม่ต้องใช้ Docker)
make test-all     # + integration ยิงใส่ Postgres จริง (testcontainers)
make sqlc         # สร้างโค้ด DB ใหม่จาก queries/ + migrations/
make vet fmt tidy
```

> ไม่มี `make`? คำสั่ง `go`/`sqlc` ตรง ๆ อยู่ใน `Makefile` ทั้งหมด

## โครง

| ที่ | ทำอะไร |
|---|---|
| `cmd/api` | boot: config → pgxpool (retry) → migrate → seed (ถ้าเป็นเดโม) → gin → graceful shutdown |
| `cmd/seed` | ติดตั้งครั้งแรกสำหรับร้านจริง — ไม่แตะฐานข้อมูลที่มีข้อมูลแล้ว เว้นแต่สั่ง `-reset` |
| `internal/config` | โหลด/ตรวจ env ตอนบูต ผิดเมื่อไหร่ process ไม่ start (fail fast) |
| `internal/dbmigrate` | รัน migration ที่ฝังไว้ ใช้ advisory lock กันหลาย replica ชนกัน |
| `internal/web` | เสิร์ฟ SPA ที่ฝังไว้ — cache header, gzip เตรียมไว้ตอนบูต, SPA fallback |
| `internal/demo` | สร้างชุดข้อมูลตัวอย่างแบบ deterministic (1 เดือน ~1,500 บิล) |
| `internal/service` | business logic — ขาย/สต็อก/รายงาน/ชำระเงิน |
| `internal/store` | **sqlc generated** อย่าแก้มือ — แก้ `queries/*.sql` แล้ว `make sqlc` |
| `internal/auth` · `middleware` · `handler` · `receipt` · `domain` | argon2id/JWT · RBAC, rate limit, request-id · HTTP layer · ใบเสร็จ HTML+ESC/POS · error kinds |
| `migrations/` | golang-migrate (`NNNN_name.up.sql`/`.down.sql`) + `embed.go` ที่เปิดให้ไบนารีอ่าน |

## Config

ตารางเต็มอยู่ที่ [`docs/deploy.md`](../docs/deploy.md#ตัวแปรสภาพแวดล้อม)
ที่ต้องมีเสมอคือ `DATABASE_URL`, `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`
(production บังคับความยาว ≥ 32 ตัว)

## Migrations

แอปรัน migration ที่ค้างให้เองตอนบูต (`AUTO_MIGRATE=true`) จึงไม่มี `migrate-up`
ให้สั่ง ส่วนการถอยกลับยังต้องสั่งเอง: `make migrate-down`

ไฟล์ `.sql` ยังเป็น golang-migrate ปกติทุกอย่าง — `migrations/embed.go`
แค่เปิดให้ไบนารีอ่านโฟลเดอร์เดียวกันนี้ ไม่มี schema ฉบับที่สองให้หลุดจากกัน

| version | เพิ่มอะไร |
|---|---|
| `0001_init` | ตารางหลักทั้งหมด · เงินเป็น `BIGINT` สตางค์ · trigger กัน UPDATE/DELETE บน ledger และบิล |
| `0002_refresh_tokens` | refresh token (เก็บเฉพาะ SHA-256 hash) + rotation |
| `0003_payments` | payment intent สำหรับยืนยันเงินโอนอัตโนมัติ |
