# ตรวจเงินเข้าอัตโนมัติ (PromptPay auto-confirm)

ทำให้หน้าแคชเชียร์ **ยืนยัน "โอนจ่าย" ให้เองเมื่อเงินเข้าจริง** โดยไม่ต้องกดเอง
ฟรี ไม่มีค่าธรรมเนียม ไม่ต้องเปิด merchant กับธนาคาร

## หลักการ
```
กด "โอนจ่าย" → สร้าง payment intent (ยอด = ยอดบิล) → โชว์ QR + "รอรับเงินอัตโนมัติ…"
เงินเข้าบัญชี → [แหล่งแจ้งเตือน] → ยิง webhook เข้า backend → จับคู่ยอดกับ intent → paid
หน้าแคชเชียร์ poll ทุก 2.5 วิ เห็น paid → ✅ ยืนยัน + พิมพ์ใบเสร็จอัตโนมัติ
```
- จับคู่ด้วย **ยอดตรงเป๊ะ** ภายใน **10 นาที** (intent หมดอายุ) — เงินแม่น ไม่บวกสตางค์
- **ออฟไลน์ → กดยืนยันเองเหมือนเดิม** (ปุ่ม "เงินเข้าแล้ว · ยืนยัน" ยังอยู่เสมอ)

## ปลายทาง webhook (backend มีให้ 2 ปาก)
| Endpoint | ใครเรียก | ยืนยันตัวตน |
|---|---|---|
| `POST /api/v1/webhooks/line` | **LINE Official Account** (Messaging API) | LINE signature (`LINE_CHANNEL_SECRET`) |
| `POST /api/v1/webhooks/payment` | อะไรก็ได้ (Apps Script / มือถือ Android) | header `X-Notify-Secret` (`PAYMENT_NOTIFY_SECRET`) |

ทั้งคู่ดึง "ยอดเงิน" จากข้อความแล้วจับคู่ให้เหมือนกัน

## ตั้งค่า env (backend)
```dotenv
PAYMENT_NOTIFY_SECRET=<สุ่มยาวๆ>      # เปิดปาก /webhooks/payment
LINE_CHANNEL_SECRET=<จาก LINE console> # เปิดปาก /webhooks/line
LINE_AMOUNT_REGEX=                     # (ไม่บังคับ) regex ดึงยอดเอง 1 capture group
```

---

## วิธี 1 — LINE OA (ใช้ได้ทั้ง iOS/Android) ✅ ที่คุณต้องการ
1. LINE Developers Console → สร้าง **Messaging API channel** (ผูกกับ LINE OA ของร้าน)
2. คัดลอก **Channel secret** → ใส่ `LINE_CHANNEL_SECRET`
3. ตั้ง **Webhook URL** = `https://pos.example.com/api/v1/webhooks/line` → เปิด "Use webhook"
4. ทำให้ "ข้อความแจ้งเงินเข้า" เข้ามาใน OA นี้ (เช่น ต่อ alert ของธนาคาร/บริการที่ push เข้า LINE)
   → ทุกข้อความ **text** ที่มีตัวเลขยอด (เช่น `รับเงิน 29.00 บาท`) จะถูกจับคู่ให้อัตโนมัติ

> ระบบอ่านยอดจากข้อความ: เลือกตัวเลขที่ตามหลังคำว่า รับ/เข้า/โอน/credit/received ก่อน
> ถ้ารูปแบบข้อความแปลก ตั้ง `LINE_AMOUNT_REGEX` เองได้ (มี 1 capture group เป็นยอดบาท)

## วิธี 2 — Bridge จากอีเมลแจ้งเงินเข้า (ฟรี, cross-platform, ไม่ต้องมีมือถือเปิดค้าง)
ธนาคารส่วนใหญ่ส่ง **อีเมลแจ้งเงินเข้า** ฟรี → ใช้ **Google Apps Script** อ่านอีเมล → ยิงเข้า webhook:
```javascript
// Apps Script (ตั้ง trigger ทุก 1 นาที) — อ่านเมลแจ้งเงินเข้าแล้ว POST
function checkDeposits(){
  const threads = GmailApp.search('from:bank subject:เงินเข้า is:unread', 0, 10);
  for (const t of threads) for (const m of t.getMessages()){
    const amt = (m.getPlainBody().match(/([0-9,]+\.[0-9]{2})/)||[])[1];
    if (amt) UrlFetchApp.fetch('https://pos.example.com/api/v1/webhooks/payment', {
      method:'post', contentType:'application/json',
      headers:{'X-Notify-Secret':'<PAYMENT_NOTIFY_SECRET>'},
      payload: JSON.stringify({ amount: parseFloat(amt.replace(/,/g,'')) })
    });
    m.markRead();
  }
}
```

## วิธี 3 — Android notification (เฉพาะ Android)
มือถือ Android เปิดแอปแบงก์ + แอป forwarder (MacroDroid/Tasker) → เจอ noti เงินเข้า → POST เข้า
`/webhooks/payment` พร้อม header `X-Notify-Secret` และ body `{"amount": 29.00}`
> ข้อจำกัด: ใช้ iOS ไม่ได้ (จึงแนะนำวิธี 1 หรือ 2)

## ทดสอบ (ยิงมือ)
```bash
curl -X POST https://pos.example.com/api/v1/webhooks/payment \
  -H 'X-Notify-Secret: <secret>' -H 'Content-Type: application/json' \
  -d '{"amount": 29.00}'          # → {"matched": true} ถ้ามีบิลรอยอดนี้
```

## ข้อควรรู้
- จับคู่ **ยอดตรง + ช่วงเวลา** → ถ้า 2 บิลยอดเท่ากันจ่ายพร้อมกันเป๊ะๆ อาจสลับได้ (โอกาสน้อยมากในร้านเล็ก) — จับคู่บิลที่เปิดก่อน (FIFO)
- ต้อง **HTTPS** (เหมือนทั้งระบบ)
- ปาก webhook มี **rate limit** + secret/signature ป้องกันคนยิงมั่ว
