# Deploy

Sabai POS ทั้งระบบคือ **คอนเทนเนอร์เดียว** — Go binary ที่ฝังหน้าเว็บไว้ข้างใน
รัน migration ให้ตัวเองตอนบูต และเสิร์ฟทั้ง `/` และ `/api` บนพอร์ตเดียว
สิ่งเดียวที่ต้องมีเพิ่มคือ PostgreSQL

เอกสารนี้มี 2 เส้นทาง เลือกอันที่ตรงกับสถานการณ์:

- [A. Azure Container Apps](#a-azure-container-apps) — เส้นทางของเดโมสาธารณะ มี CI/CD และ scale-to-zero
- [B. เซิร์ฟเวอร์ของตัวเอง (VPS)](#b-เซิร์ฟเวอร์ของตัวเอง-vps) — เส้นทางของร้านที่ซื้อไปใช้จริง

---

## ตัวแปรสภาพแวดล้อม

| ตัวแปร | จำเป็น | ค่าเริ่มต้น | ความหมาย |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | `postgres://user:pass@host:5432/db?sslmode=require` |
| `JWT_ACCESS_SECRET` | ✅ | — | สุ่มมา ≥ 32 ตัวอักษร (production บังคับ) |
| `JWT_REFRESH_SECRET` | ✅ | — | คนละค่ากับข้างบน |
| `APP_ENV` | | `development` | `production` เปิด `Secure` cookie และปิด debug log |
| `HTTP_PORT` | | `8080` | |
| `APP_VERSION` | | `dev` | โชว์ที่ `/api/v1/meta` ใช้ยืนยันว่า deploy ติดจริง |
| `AUTO_MIGRATE` | | `true` | รัน migration ที่ค้างตอนบูต |
| `SERVE_UI` | | `true` | เสิร์ฟหน้าเว็บที่ฝังไว้ (`false` = API อย่างเดียว) |
| `DEMO_MODE` | | `false` | ⚠️ ดูหัวข้อถัดไป |
| `DEMO_RESET_EVERY` | | `24h` | อายุสูงสุดของชุดข้อมูลตัวอย่างก่อนถูกสร้างใหม่ (`0` = ไม่ทำ) นับจากเวลาที่บันทึกไว้ในฐานข้อมูล ไม่ใช่ uptime ของโปรเซส — จำเป็นเมื่อแอป scale-to-zero |
| `PAYMENT_NOTIFY_SECRET` | | ว่าง | ว่าง = ปิด `/webhooks/payment` |
| `LINE_CHANNEL_SECRET` | | ว่าง | ว่าง = ปิด `/webhooks/line` |

### ⚠️ `DEMO_MODE` ต้องเป็น `false` สำหรับร้านจริง

เปิดไว้แล้วจะเกิด 3 อย่าง: ข้อมูลตัวอย่างถูกสร้างลงฐานข้อมูลที่ว่าง,
รหัสผ่านทุกบัญชีถูกประกาศที่ `/api/v1/meta`, และ `POST /api/v1/demo/reset`
ถูกเปิดให้ใครก็ได้ยิงเพื่อล้างข้อมูลทั้งหมด
`docker-compose.prod.yml` บังคับเป็น `false` ไว้ให้แล้ว

---

## A. Azure Container Apps

โครงที่ใช้จริงกับเดโมตัวนี้ ค่าใช้จ่ายจริงอยู่ที่ประมาณศูนย์:
Container Apps มีโควตาฟรีรายเดือน (180,000 vCPU-วินาที) และแอปตั้ง
`min-replicas 0` ไว้ จึงไม่เสียค่าอะไรตอนไม่มีคนเข้า ฐานข้อมูลใช้ Neon free tier

### 1. ฐานข้อมูล (Neon)

สร้างโปรเจกต์ที่ [neon.tech](https://neon.tech) เลือก region ที่ใกล้แอป
(Singapore สำหรับ `southeastasia`) แล้วคัดลอก connection string
ต้องมี `?sslmode=require` ต่อท้าย

### 2. สร้างทรัพยากรครั้งเดียว

```bash
az group create -n rg-sabai-pos -l southeastasia
az containerapp env create -n cae-sabai-pos -g rg-sabai-pos \
  -l southeastasia --logs-destination none

az containerapp create \
  -n sabai-pos -g rg-sabai-pos --environment cae-sabai-pos \
  --image ghcr.io/<owner>/sabai-pos:latest \
  --target-port 8080 --ingress external \
  --min-replicas 0 --max-replicas 2 \
  --cpu 0.5 --memory 1Gi \
  --secrets db-url="postgres://…?sslmode=require" \
            jwt-access="$(openssl rand -hex 32)" \
            jwt-refresh="$(openssl rand -hex 32)" \
  --env-vars APP_ENV=production DEMO_MODE=true \
             DATABASE_URL=secretref:db-url \
             JWT_ACCESS_SECRET=secretref:jwt-access \
             JWT_REFRESH_SECRET=secretref:jwt-refresh
```

`--min-replicas 0` คือหัวใจของค่าใช้จ่าย: แอปดับตัวเองเมื่อไม่มีทราฟฟิก
คำขอแรกหลังจากนั้นจะช้ากว่าปกติ (~2 วินาที) เพราะทั้งคอนเทนเนอร์และ Neon
ต้องตื่นพร้อมกัน — โค้ดฝั่ง Go จึง retry การต่อฐานข้อมูลแทนที่จะตายไปเลย

### 3. CI/CD (`.github/workflows/deploy.yml`)

push เข้า `main` → build image → push `ghcr.io` → `az containerapp update`
→ **รอจน `/readyz` ตอบและ `/api/v1/meta` รายงานเวอร์ชันใหม่** ถึงจะถือว่าสำเร็จ

Azure ยืนยันตัวตนด้วย OIDC ไม่มีรหัสผ่านเก็บในรีโป — GitHub ขอ token
อายุสั้นที่ Azure เชื่อถือเฉพาะรีโปนี้และ branch นี้เท่านั้น

ตั้งครั้งเดียว:

```bash
appId=$(az ad app create --display-name sabai-pos-github-deploy --query appId -o tsv)
az ad sp create --id "$appId"
az role assignment create --assignee "$appId" --role Contributor \
  --scope "/subscriptions/<sub>/resourceGroups/rg-sabai-pos"   # แค่ RG นี้ ไม่ใช่ทั้ง subscription

az ad app federated-credential create --id "$appId" --parameters '{
  "name": "github-main",
  "issuer": "https://token.actions.githubusercontent.com",
  "subject": "repo:<owner>/<repo>:ref:refs/heads/main",
  "audiences": ["api://AzureADTokenExchange"]
}'

gh secret set AZURE_CLIENT_ID       --body "$appId"
gh secret set AZURE_TENANT_ID       --body "$(az account show --query tenantId -o tsv)"
gh secret set AZURE_SUBSCRIPTION_ID --body "$(az account show --query id -o tsv)"
```

image บน GHCR ต้องเป็น **public** เพื่อให้ Container Apps ดึงได้โดยไม่ต้องมี
credential ค้างไว้ (Settings → Packages → เลือก package → Change visibility)
ถ้าจำเป็นต้องเก็บเป็น private ให้ผูก credential ที่ฝั่ง Container Apps แทน:
`az containerapp registry set -n sabai-pos -g rg-sabai-pos --server ghcr.io --username <user> --password <PAT ที่มี read:packages>`

### 4. โดเมนของตัวเอง (ถ้ามี)

```bash
az containerapp hostname add -n sabai-pos -g rg-sabai-pos --hostname pos.example.com
az containerapp hostname bind -n sabai-pos -g rg-sabai-pos --hostname pos.example.com \
  --environment cae-sabai-pos --validation-method CNAME
```
Container Apps ออกและต่ออายุใบรับรองให้ฟรี

---

## B. เซิร์ฟเวอร์ของตัวเอง (VPS)

สำหรับร้านที่ต้องการเก็บข้อมูลไว้กับตัวเอง

**สิ่งที่ต้องมี** — VPS (Ubuntu 22.04+) 1 vCPU / 1–2 GB RAM,
Docker + Compose plugin, และโดเมนที่ชี้มาที่เครื่อง
(HTTPS จำเป็น เพราะ refresh cookie เป็น `Secure`)

```bash
git clone <repo> sabai-pos && cd sabai-pos
cp .env.example .env
openssl rand -hex 32    # → JWT_ACCESS_SECRET
openssl rand -hex 32    # → JWT_REFRESH_SECRET
```

```dotenv
POSTGRES_USER=sabai
POSTGRES_PASSWORD=<รหัสยาว ๆ>
POSTGRES_DB=sabai
JWT_ACCESS_SECRET=<hex 32 ไบต์>
JWT_REFRESH_SECRET=<hex 32 ไบต์>
WEB_PORT=8090          # พอร์ตภายใน Caddy จะ proxy มาที่นี่
```

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml run --rm --entrypoint /seed app
```

> ⚠️ หลัง seed ให้ **เปลี่ยนรหัสผ่าน** ของ owner/manager/cashier ทันที
> ผ่าน `POST /api/v1/auth/change-password`

### HTTPS ด้วย Caddy

`/etc/caddy/Caddyfile`
```
pos.example.com {
    reverse_proxy localhost:8090
}
```
```bash
sudo apt install caddy && sudo systemctl reload caddy
```

### สำรองข้อมูล

```bash
./scripts/backup.sh    # → backups/sabai-pos-YYYYmmdd-HHMMSS.sql.gz (เก็บ 14 ชุดล่าสุด)
```
cron ทุกวันตี 2:
```cron
0 2 * * * cd /path/sabai-pos && ./scripts/backup.sh >> backups/backup.log 2>&1
```
กู้คืน (เขียนทับ — ระวัง): `./scripts/restore.sh backups/sabai-pos-….sql.gz`

### อัปเดต

```bash
git pull
docker compose -f docker-compose.prod.yml up -d --build
```
migration รันเองตอนบูต เป็นแบบ forward-only versioned (golang-migrate)
ปลอดภัยกับข้อมูลเดิม · rollback แอป: `git checkout <tag เดิม>` แล้ว up ใหม่
(ระวัง migration ที่ลงไปแล้ว — ถอยด้วย `make migrate-down`)

---

## ตรวจสุขภาพ

```bash
curl -f https://<host>/healthz          # {"status":"ok"}         — liveness
curl -f https://<host>/readyz           # {"db":"up",…}           — เช็ค DB ด้วย
curl -s https://<host>/api/v1/meta      # ยืนยันเวอร์ชันที่กำลังเสิร์ฟ
```

ต่อ uptime monitor เข้า `/readyz` ได้เลย ถ้าใช้ scale-to-zero ให้ตั้ง timeout
ของ monitor ไว้ที่ 30 วินาทีขึ้นไป เพราะคำขอแรกต้องปลุกทั้งคอนเทนเนอร์และฐานข้อมูล

ดู log:
```bash
az containerapp logs show -n sabai-pos -g rg-sabai-pos --follow   # Azure
docker compose -f docker-compose.prod.yml logs -f app             # VPS
```
