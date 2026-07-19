// Package receipt renders a sale into a printable receipt: HTML (reliable,
// browser/print-server) and raw ESC/POS bytes (direct thermal 58/80mm).
// All money is integer satang; formatting to baht is display-only.
package receipt

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type Line struct {
	Name      string
	Qty       int32
	Price     int64 // satang
	LineTotal int64 // satang
}

type View struct {
	ShopName      string
	BillNo        string
	Time          string
	Cashier       string
	Items         []Line
	Subtotal      int64
	Discount      int64
	Total         int64
	Paid          int64
	Change        int64
	PaymentMethod string // cash | transfer
	Voided        bool
}

// Money formats satang as a grouped baht string ("12,345.50") using integer
// math only — no float ever touches a money value.
func Money(satang int64) string {
	neg := satang < 0
	if neg {
		satang = -satang
	}
	baht := satang / 100
	cents := satang % 100
	s := fmt.Sprintf("%d", baht)
	// insert thousands separators
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	res := fmt.Sprintf("%s.%02d", out, cents)
	if neg {
		return "-" + res
	}
	return res
}

func methodLabel(m string) string {
	if m == "transfer" {
		return "โอน/พร้อมเพย์"
	}
	return "เงินสด"
}

func widthChars(mm int) int {
	if mm == 80 {
		return 48
	}
	return 32
}

// ── HTML ─────────────────────────────────────────────────────────────────────

func HTML(v View, mm int) string {
	maxW := 300
	if mm != 80 {
		maxW = 220
	}
	var b strings.Builder
	esc := html.EscapeString
	b.WriteString(`<!doctype html><html lang="th"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	b.WriteString(`<title>` + esc(v.BillNo) + `</title><style>`)
	b.WriteString(`*{box-sizing:border-box}body{margin:0;background:#eee;font-family:'TH Sarabun New','Sarabun',monospace}`)
	b.WriteString(fmt.Sprintf(`.r{max-width:%dpx;margin:0 auto;background:#fff;padding:10px 12px;font-size:13px;color:#000}`, maxW))
	b.WriteString(`.c{text-align:center}.b{font-weight:700}.row{display:flex;justify-content:space-between;gap:8px}`)
	b.WriteString(`.hr{border-top:1px dashed #000;margin:6px 0}.mut{color:#444;font-size:12px}.big{font-size:16px}`)
	b.WriteString(`.void{color:#a32617;border:1px solid #a32617;padding:1px 6px;border-radius:4px;display:inline-block}`)
	b.WriteString(`@media print{body{background:#fff}.r{box-shadow:none}.noprint{display:none}}`)
	b.WriteString(`</style></head><body><div class="r">`)
	b.WriteString(`<div class="c b big">` + esc(v.ShopName) + `</div>`)
	b.WriteString(`<div class="c mut">ใบเสร็จรับเงิน (อย่างย่อ)</div>`)
	if v.Voided {
		b.WriteString(`<div class="c" style="margin:4px 0"><span class="void b">ยกเลิกแล้ว</span></div>`)
	}
	b.WriteString(`<div class="c mut">` + esc(v.BillNo) + ` · ` + esc(v.Time) + `</div>`)
	if v.Cashier != "" {
		b.WriteString(`<div class="c mut">แคชเชียร์: ` + esc(v.Cashier) + `</div>`)
	}
	b.WriteString(`<div class="hr"></div>`)
	for _, it := range v.Items {
		b.WriteString(`<div class="row"><span>` + esc(it.Name) + `</span></div>`)
		b.WriteString(`<div class="row mut"><span>` + fmt.Sprintf("%d × %s", it.Qty, Money(it.Price)) +
			`</span><span>` + Money(it.LineTotal) + `</span></div>`)
	}
	b.WriteString(`<div class="hr"></div>`)
	if v.Discount > 0 {
		b.WriteString(`<div class="row"><span>ยอดก่อนลด</span><span>` + Money(v.Subtotal) + `</span></div>`)
		b.WriteString(`<div class="row"><span>ส่วนลด</span><span>-` + Money(v.Discount) + `</span></div>`)
	}
	b.WriteString(`<div class="row b big"><span>รวมทั้งสิ้น</span><span>` + Money(v.Total) + `</span></div>`)
	b.WriteString(`<div class="row mut"><span>` + esc(methodLabel(v.PaymentMethod)) + `</span><span>` + Money(v.Paid) + `</span></div>`)
	if v.Change > 0 {
		b.WriteString(`<div class="row mut"><span>เงินทอน</span><span>` + Money(v.Change) + `</span></div>`)
	}
	b.WriteString(`<div class="hr"></div>`)
	b.WriteString(`<div class="c mut">ขอบคุณที่อุดหนุนจ้า 🙏</div>`)
	b.WriteString(`<div class="c noprint" style="margin-top:10px"><button onclick="print()">พิมพ์</button></div>`)
	b.WriteString(`</div></body></html>`)
	return b.String()
}

// ── ESC/POS ──────────────────────────────────────────────────────────────────

var (
	escInit     = []byte{0x1B, 0x40}       // ESC @
	escCodepage = []byte{0x1B, 0x74, 0x15} // ESC t 21 — Thai (CP874/TIS-620); adjust per printer
	alignLeft   = []byte{0x1B, 0x61, 0x00}
	alignCenter = []byte{0x1B, 0x61, 0x01}
	boldOn      = []byte{0x1B, 0x45, 0x01}
	boldOff     = []byte{0x1B, 0x45, 0x00}
	doubleOn    = []byte{0x1D, 0x21, 0x11} // GS ! double width+height
	doubleOff   = []byte{0x1D, 0x21, 0x00}
	feedAndCut  = []byte{0x0A, 0x0A, 0x0A, 0x1D, 0x56, 0x42, 0x00} // feed + partial cut
	thaiEncoder = charmap.Windows874.NewEncoder()
)

func enc(s string) []byte {
	b, _, err := transform.Bytes(thaiEncoder, []byte(s))
	if err != nil {
		return []byte(s) // fall back to raw UTF-8 if a rune can't map
	}
	return b
}

// leftRight pads left/right text to the paper width.
func leftRight(left, right string, w int) string {
	space := w - len([]rune(left)) - len([]rune(right))
	if space < 1 {
		space = 1
	}
	return left + strings.Repeat(" ", space) + right
}

// ESCPOS renders the receipt as raw printer bytes (Thai via CP874).
func ESCPOS(v View, mm int) []byte {
	w := widthChars(mm)
	var b bytes.Buffer
	b.Write(escInit)
	b.Write(escCodepage)

	b.Write(alignCenter)
	b.Write(boldOn)
	b.Write(doubleOn)
	b.Write(enc(v.ShopName))
	b.Write([]byte{0x0A})
	b.Write(doubleOff)
	b.Write(boldOff)
	b.Write(enc("ใบเสร็จรับเงิน (อย่างย่อ)"))
	b.Write([]byte{0x0A})
	if v.Voided {
		b.Write(boldOn)
		b.Write(enc("*** ยกเลิกแล้ว ***"))
		b.Write(boldOff)
		b.Write([]byte{0x0A})
	}
	b.Write(enc(v.BillNo + " " + v.Time))
	b.Write([]byte{0x0A})

	b.Write(alignLeft)
	b.Write(enc(strings.Repeat("-", w)))
	b.Write([]byte{0x0A})
	for _, it := range v.Items {
		b.Write(enc(it.Name))
		b.Write([]byte{0x0A})
		b.Write(enc(leftRight(fmt.Sprintf("  %d x %s", it.Qty, Money(it.Price)), Money(it.LineTotal), w)))
		b.Write([]byte{0x0A})
	}
	b.Write(enc(strings.Repeat("-", w)))
	b.Write([]byte{0x0A})
	if v.Discount > 0 {
		b.Write(enc(leftRight("ยอดก่อนลด", Money(v.Subtotal), w)))
		b.Write([]byte{0x0A})
		b.Write(enc(leftRight("ส่วนลด", "-"+Money(v.Discount), w)))
		b.Write([]byte{0x0A})
	}
	b.Write(boldOn)
	b.Write(enc(leftRight("รวมทั้งสิ้น", Money(v.Total), w)))
	b.Write(boldOff)
	b.Write([]byte{0x0A})
	b.Write(enc(leftRight(methodLabel(v.PaymentMethod), Money(v.Paid), w)))
	b.Write([]byte{0x0A})
	if v.Change > 0 {
		b.Write(enc(leftRight("เงินทอน", Money(v.Change), w)))
		b.Write([]byte{0x0A})
	}
	b.Write(alignCenter)
	b.Write(enc("ขอบคุณที่อุดหนุน"))
	b.Write(feedAndCut)
	return b.Bytes()
}
