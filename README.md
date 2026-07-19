# MiniMart POS 🛒

ระบบ Point-of-Sale ร้านมินิมาร์ท — **offline-first, multi-tenant-ready, production-grade**
Backend: Go + PostgreSQL · Frontend: Vite + React (PWA)

> เอกสารนี้คือ root ของ monorepo งานแบ่งเป็น 4 phase ตาม
> [build brief](docs/) — ตอนนี้อยู่ปลาย **Phase 1**

## โครงสร้าง

```
minimart-pos/
├── backend/           Go 1.25 · Gin · PostgreSQL 16 · sqlc · JWT · Zap
│   ├── cmd/api/            entrypoint (config → logger → db → http + graceful shutdown)
│   ├── internal/
│   │   ├── config/         โหลด+ตรวจ env, fail fast
│   │   ├── logger/         Zap
│   │   ├── middleware/     request-id · logging · recovery · CORS
│   │   ├── handler/        gin handlers (บาง) — /healthz /readyz
│   │   └── store/          sqlc generated (type-safe queries)
│   ├── migrations/         golang-migrate (0001_init = schema เต็ม)
│   ├── queries/            .sql สำหรับ sqlc
│   └── Dockerfile          multi-stage, distroless non-root
├── frontend/          Vite + React + TS (POS client — rebuild เต็มใน Phase 3)
├── docs/
│   ├── adr/                Architecture Decision Records
│   └── design/            ดีไซน์ต้นฉบับ (MiniMart-POS.dc.html)
└── docker-compose.yml  db + migrate + api (คำสั่งเดียวขึ้นครบ)
```

## เริ่มใช้งาน (dev)

ต้องมี **Docker Desktop** ทำงานอยู่

```bash
cp .env.example .env        # แล้วแก้ password/secrets ตามต้องการ
docker compose up -d --build
```

จะได้:
- Postgres (host port **5433**), รัน migration อัตโนมัติ
- API ที่ **http://localhost:8082**

```bash
curl http://localhost:8082/healthz   # {"status":"ok"}
curl http://localhost:8082/readyz    # {"db":"up","status":"ok"}
```

### ใส่ข้อมูลตัวอย่าง (seed)

```bash
cd backend
DATABASE_URL="postgres://minimart:change_me_dev_password@localhost:5433/minimart?sslmode=disable" \
  go run ./cmd/seed
```

สร้าง 1 ร้าน + 30 สินค้า + ผู้ใช้ dev (idempotent):

| ผู้ใช้ | รหัสผ่าน | บทบาท |
|---|---|---|
| `owner` | `owner1234` | superadmin |
| `manager` | `manager1234` | manager |
| `cashier` | `cashier1234` | cashier |

### API v1 (มีแล้ว)

| Method | Path | สิทธิ์ |
|---|---|---|
| POST | `/api/v1/auth/login` `/refresh` `/logout` | public (refresh ใช้ httpOnly cookie) |
| GET | `/api/v1/auth/me` | auth |
| POST | `/api/v1/auth/change-password` | auth (เปลี่ยนรหัสตัวเอง + เพิกถอน session อื่น) |
| POST | `/api/v1/auth/register` | manager+ |
| GET | `/api/v1/products` · `/products/:id` · `/products/:id/onhand` | auth (รวม cashier) |
| POST/PUT/DELETE | `/api/v1/products` … | **manager+** (cashier โดน 403) |
| POST | `/api/v1/products/:id/receive` | manager+ (idempotent ด้วย `client_uuid`) |
| POST | `/api/v1/bills` (ปิดการขาย) | auth (cashier ขายได้) · atomic · idempotent |
| GET | `/api/v1/bills/:id` | auth |
| GET | `/api/v1/bills/:id/receipt?format=html\|escpos&width=58\|80` | auth (พิมพ์ใบเสร็จ) |
| POST | `/api/v1/bills/:id/void` | **manager+** (สร้าง void bill คืนสต็อก) |

### รันเทสต์

```bash
cd backend
go test ./...                              # unit (เร็ว ไม่ต้อง Docker)
go test -tags=integration ./internal/store # integration (testcontainers → Postgres จริง)
```

## Production

deploy จริงด้วย `docker-compose.prod.yml` (nginx เสิร์ฟ SPA + proxy `/api`, backend/db ภายใน):

```bash
cp .env.example .env    # ตั้ง secret แข็งแรง + โดเมน
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml run --rm --entrypoint /seed backend   # ครั้งแรก
```

- **HTTPS จำเป็น** (refresh cookie เป็น Secure) — ใช้ Caddy ด้านหน้า → [docs/deploy.md](docs/deploy.md)
- **CI**: `.github/workflows/ci.yml` (lint+test+build) · images บน tag: `release.yml`
- **สำรองข้อมูล**: `scripts/backup.sh` (+ cron) · กู้คืน: `scripts/restore.sh`
- ส่งมอบ/ดูแล: [docs/handover.md](docs/handover.md)

> พอร์ต 5433/8082 ถูกเลือกเพื่อเลี่ยงการชนกับ Postgres/บริการอื่นบนเครื่องนี้
> (แก้ได้ใน `docker-compose.yml`) การเชื่อมภายใน container ใช้ `db:5432` เสมอ

### รัน backend ตรงๆ (ไม่ผ่าน compose)

```bash
cd backend
cp .env.example .env        # ชี้ DATABASE_URL ไปที่ Postgres ที่รันอยู่
go run ./cmd/api            # หรือ: make run
```

## หลักการสำคัญ (ห้ามพลาด)

- **เงิน = integer satang** (1 บาท = 100 สตางค์) ไม่มี float ทั้งระบบ → [ADR 0001](docs/adr/0001-money-representation.md)
- **สต็อก = append-only ledger** (`stock_movements`) แก้/ลบไม่ได้ (มี trigger กัน)
- **บิล immutable** + snapshot ราคา, `client_uuid` เป็น idempotency key สำหรับ offline sync
- ทุกตารางมี `store_id` (multi-tenant-ready) → [ADR 0002](docs/adr/0002-architecture-and-stack.md)

## สถานะงาน

| Phase | งาน | สถานะ |
|---|---|---|
| 1 | docker-compose + โครงโปรเจกต์ + migration แรก | ✅ เสร็จ |
| 1 | sqlc + auth (argon2id/JWT/refresh/RBAC) + CRUD สินค้า/รับสต็อก + seed + test | ✅ เสร็จ |
| 2 | ปิดการขาย (atomic/idempotent) + เลขบิล gap-free + void/refund + ใบเสร็จ HTML/ESC-POS | ✅ เสร็จ |
| 3a | report endpoints + frontend offline-first (Router/Dexie/sync/PWA) + Login + Cashier | ✅ เสร็จ |
| 3b | หน้าร้าน + สต็อก(รับเข้า/CRUD, RBAC) + แดชบอร์ด(reports) + จอลูกค้า + ยิงบาร์โค้ด | ✅ เสร็จ |
| 4 | prod image (nginx) + CI/CD + backup + deploy/handover docs | ✅ **เสร็จ — พร้อมขาย** |
| 4 | Deploy/CI-CD/backup/handover docs | ⬜ |
