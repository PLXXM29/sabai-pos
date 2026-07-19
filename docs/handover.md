# Handover — MiniMart POS

เอกสารส่งมอบ: ภาพรวมระบบ วิธีดูแล และงานที่พบบ่อย

## ระบบคืออะไร
POS ร้านมินิมาร์ท **offline-first**: เครื่องแคชเชียร์ขายได้แม้เน็ตหลุด แล้ว sync ทีหลัง
เงินทุกจุดเก็บเป็น **integer satang** (ไม่มี float) · บิล/ledger สต็อก **แก้ไม่ได้** (immutable)
ออกแบบเผื่อหลายสาขา (`store_id` ทุกตาราง) แต่รันร้านเดียวก่อน

## สถาปัตยกรรม
```
[Frontend PWA (React)]  ──/api──►  [nginx] ──►  [Backend Go/Gin]  ──►  [PostgreSQL]
  Dexie/IndexedDB (local-first)                   handler→service→store(sqlc)
  sync engine (pending queue, client_uuid)        JWT + refresh cookie + RBAC
```
รายละเอียดการตัดสินใจ: `docs/adr/` · โครง/คำสั่ง: `README.md`, `backend/README.md`, `frontend/README.md`

## บทบาทผู้ใช้ (RBAC — บังคับที่ backend)
| บทบาท | ทำได้ |
|---|---|
| cashier | ขาย, ดูสินค้า/สต็อก (อ่านอย่างเดียว), พิมพ์ใบเสร็จ |
| manager | ทุกอย่างของ cashier + เพิ่ม/แก้/ลบ/รับเข้าสินค้า, ยกเลิกบิล (void), ดูแดชบอร์ด/กำไร |
| superadmin | เหมือน manager (เผื่อสิทธิ์ระดับสูงในอนาคต) |

## งานที่พบบ่อย

**เพิ่มพนักงาน** — ให้ manager login แล้วเรียก (หรือทำผ่าน UI เมื่อมีหน้าจอผู้ใช้):
```bash
curl -X POST https://pos.example.com/api/v1/auth/register \
  -H "Authorization: Bearer <manager token>" -H 'Content-Type: application/json' \
  -d '{"username":"somchai","password":"xxxxxx","role":"cashier"}'
```

**เปลี่ยนรหัสผ่านตัวเอง** (ผู้ใช้ login แล้วเปลี่ยนเอง — session อื่นจะถูกเพิกถอน):
```bash
curl -X POST https://pos.example.com/api/v1/auth/change-password \
  -H "Authorization: Bearer <token>" -H 'Content-Type: application/json' \
  -d '{"current_password":"<เดิม>","new_password":"<ใหม่ ≥6 ตัว>"}'
```
> หลัง seed ให้ owner/manager/cashier เปลี่ยนรหัส default ทันทีด้วยวิธีนี้

**เปิดร้านใหม่ (สาขาใหม่)** — seed สร้าง 1 ร้าน; หลายร้านให้เพิ่ม store + users เพิ่ม
(ระบบเผื่อ `store_id` ไว้แล้ว แต่ยังไม่มี UI จัดการหลายร้าน — งานต่อยอด)

**ยกเลิกบิล (คืนเงิน)** — manager: `POST /api/v1/bills/{id}/void` → สร้างบิล void ใหม่
คืนสต็อกให้อัตโนมัติ บิลเดิมไม่ถูกลบ (audit ครบ)

**พิมพ์ใบเสร็จซ้ำ** — `GET /api/v1/bills/{id}/receipt?format=html` (หรือ `escpos` สำหรับเครื่องพิมพ์ความร้อน)

## ดูแลรักษา
- **สำรองข้อมูล**: cron `scripts/backup.sh` ทุกวัน + ทดสอบ restore เป็นระยะ (ดู deploy.md)
- **อัปเดต**: `git pull && docker compose -f docker-compose.prod.yml up -d --build`
- **ล็อก**: `docker compose -f docker-compose.prod.yml logs -f backend` (structured JSON, มี request_id)
- **สุขภาพ**: `/healthz`, `/readyz` — ต่อ uptime monitor

## แก้ปัญหาเบื้องต้น
| อาการ | สาเหตุ/ทางแก้ |
|---|---|
| login แล้วเด้งออก / refresh ไม่ทำงาน | ต้องเสิร์ฟผ่าน **HTTPS** (refresh cookie เป็น Secure) — เช็ค Caddy/โดเมน |
| แคชเชียร์เห็น "ออฟไลน์" | เน็ตหลุด — ยังขายได้ บิลจะเข้าคิว sync เมื่อกลับมาออนไลน์ |
| บิลค้าง "รอ sync" | ดู log backend; ถ้า server ปฏิเสธ (เช่นสต็อกไม่พอจากอีกเครื่อง) บิลจะถูก mark error |
| backend ไม่ start | config ผิด (fail fast) — อ่าน log จะบอกว่า env ตัวไหนขาด/ผิด |
| เลขบิลต้องไม่ซ้ำ/ไม่ข้าม | ออกจาก `bill_counters` ในทรานแซกชันเดียว — ข้ามเฉพาะกรณี rollback (ตั้งใจ) |

## จุดที่ยังต่อยอดได้ (ไม่บล็อกการใช้งาน)
- หน้าจอจัดการผู้ใช้ + จัดการหลายสาขา (backend รองรับ store_id แล้ว)
- import สินค้าแบบวิซาร์ด (สแกน/CSV/PO) ตามดีไซน์เดิม (ตอนนี้เพิ่ม/รับเข้าทีละรายการ)
- cost snapshot ต่อบิล (กำไรตอนนี้ใช้ทุนปัจจุบัน = "โดยประมาณ")
- พักบิล/เรียกบิลพัก ฝั่ง sync (มีในดีไซน์เดิม)
