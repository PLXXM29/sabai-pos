# Deploy guide — single VPS

คู่มือติดตั้ง MiniMart POS บนเซิร์ฟเวอร์เดียว (VPS) ด้วย Docker Compose

## สิ่งที่ต้องมี
- VPS (Ubuntu 22.04+), 1 vCPU / 1–2 GB RAM พอสำหรับร้านเดียว
- **Docker + Docker Compose plugin**
- โดเมนชี้มาที่ VPS (สำหรับ HTTPS — จำเป็น เพราะ refresh cookie เป็น `Secure`)

## 1. เตรียมโค้ด + ตั้งค่า

```bash
git clone <repo> minimart && cd minimart
cp .env.example .env
```

แก้ `.env` — **สำคัญ**: ตั้ง secret ให้แข็งแรง (prod บังคับ ≥ 32 ตัว)

```bash
# สุ่ม secret
openssl rand -hex 32   # ใส่ JWT_ACCESS_SECRET
openssl rand -hex 32   # ใส่ JWT_REFRESH_SECRET
```

```dotenv
POSTGRES_USER=minimart
POSTGRES_PASSWORD=<รหัสยาว ๆ>
POSTGRES_DB=minimart
JWT_ACCESS_SECRET=<hex 32 ไบต์>
JWT_REFRESH_SECRET=<hex 32 ไบต์>
CORS_ALLOWED_ORIGINS=https://pos.example.com
WEB_PORT=8090          # พอร์ตภายในของ nginx (Caddy จะ proxy มาที่นี่)
```

## 2. ขึ้นระบบ

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

ได้: `db` (ภายใน) → `migrate` (รัน schema) → `backend` (ภายใน) → `frontend` (nginx เสิร์ฟ SPA + proxy `/api`) เปิดที่ `:8090`

### ใส่ข้อมูลร้าน + ผู้ใช้ครั้งแรก (seed)

```bash
docker compose -f docker-compose.prod.yml run --rm --entrypoint /seed backend
```

> ⚠️ หลัง seed ให้ **เปลี่ยนรหัสผ่าน** ผู้ใช้ตัวอย่าง (owner/manager/cashier) ทันที
> (หรือแก้ seed ให้ตั้งรหัสของร้านเอง)

## 3. HTTPS (จำเป็น) — Caddy ด้านหน้า

refresh cookie เป็น `Secure` จึงเดินทางเฉพาะ HTTPS ใช้ **Caddy** ทำ TLS อัตโนมัติ:

`/etc/caddy/Caddyfile`
```
pos.example.com {
    reverse_proxy localhost:8090
}
```
```bash
sudo apt install caddy && sudo systemctl reload caddy
```
Caddy จะขอใบรับรอง Let's Encrypt ให้เอง → เปิด `https://pos.example.com` ใช้งานได้

## 4. สำรองข้อมูล (backup)

```bash
./scripts/backup.sh          # → backups/minimart-YYYYmmdd-HHMMSS.sql.gz (เก็บ 14 ชุดล่าสุด)
```
ตั้ง cron ทุกวันตี 2:
```cron
0 2 * * * cd /path/minimart && ./scripts/backup.sh >> backups/backup.log 2>&1
```
กู้คืน (ระวัง — เขียนทับ):
```bash
./scripts/restore.sh backups/minimart-YYYYmmdd-HHMMSS.sql.gz
```

## 5. อัปเดตเวอร์ชันใหม่

```bash
git pull
docker compose -f docker-compose.prod.yml up -d --build   # migrate รันอัตโนมัติ
```
migration เป็น forward-only versioned (golang-migrate) — ปลอดภัยกับข้อมูลเดิม
Rollback แอป: `git checkout <tag เดิม>` แล้ว up ใหม่ (ระวัง migration ที่ลงไปแล้ว)

## 6. ตรวจสุขภาพ

```bash
curl -f https://pos.example.com/api/v1/../..   # ผ่าน Caddy
docker compose -f docker-compose.prod.yml exec backend /api --help 2>/dev/null || true
docker compose -f docker-compose.prod.yml logs -f backend
```
backend มี `/healthz` (liveness) และ `/readyz` (เช็ค DB) — ต่อ uptime monitor ได้

## ตรวจแล้วว่าใช้ได้ (2026-07-19)
ขึ้น prod stack + seed + login ผ่าน nginx ได้ token + ดึง 30 สินค้า + reports + backup gzip สำเร็จ
