# MiniMart POS — Backend

Go 1.25 · Gin · PostgreSQL 16 · sqlc · JWT · Zap

## คำสั่งที่ใช้บ่อย

```bash
go run ./cmd/api      # รัน API (make run)
go build ./...        # build
go vet ./...          # vet
go test ./...         # test (make test)
sqlc generate         # สร้างโค้ด DB จาก queries/ + migrations/ (make sqlc)
go mod tidy           # จัดการ deps (make tidy)
```

> ไม่มี `make`? ใช้คำสั่ง `go`/`sqlc` ตรงๆ ได้เลย (ดูใน `Makefile`)

## โครง

- `cmd/api/main.go` — boot: config → logger → pgxpool → gin → graceful shutdown
- `internal/config` — โหลด/ตรวจ env ตอน boot (fail fast)
- `internal/middleware` — request-id, structured logging, recovery, CORS
- `internal/handler` — gin handlers บางๆ (`/healthz`, `/readyz`)
- `internal/store` — **sqlc generated** (อย่าแก้มือ — แก้ `queries/*.sql` แล้ว `sqlc generate`)
- `migrations/` — golang-migrate (`NNNN_name.up.sql` / `.down.sql`)
- `queries/` — SQL สำหรับ sqlc

## Config (env)

| ตัวแปร | จำเป็น | ค่าเริ่มต้น |
|---|---|---|
| `DATABASE_URL` | ✅ | — |
| `JWT_ACCESS_SECRET` | ✅ | — (prod ต้อง ≥ 32 ตัว) |
| `JWT_REFRESH_SECRET` | ✅ | — (prod ต้อง ≥ 32 ตัว) |
| `APP_ENV` | | `development` |
| `HTTP_PORT` | | `8080` |
| `LOG_LEVEL` | | `info` |
| `CORS_ALLOWED_ORIGINS` | | `http://localhost:5173` |

ถ้า config ไม่ครบ/ผิด → process ไม่ start และพิมพ์บอกว่าอะไรผิด (fail fast)

## Migrations

รันอัตโนมัติผ่าน `docker compose` (service `migrate`) หรือด้วยมือ:

```bash
make migrate-up      # / migrate-down
```

Schema ปัจจุบัน: `0001_init` — สร้างครบทุกตาราง (stores, users, products, inventory,
stock_movements, bills, bill_items, bill_counters, sync_log, audit_log)
เงินเป็น `BIGINT` satang, ledger/บิล append-only (มี trigger กัน UPDATE/DELETE)
