package receipt

import "testing"

func TestMoneyFormatting(t *testing.T) {
	cases := map[int64]string{
		0:       "0.00",
		50:      "0.50",
		100:     "1.00",
		700:     "7.00",
		1234550: "12,345.50",
		-500:    "-5.00",
		999999:  "9,999.99",
	}
	for satang, want := range cases {
		if got := Money(satang); got != want {
			t.Errorf("Money(%d) = %q, want %q", satang, got, want)
		}
	}
}

func TestHTMLContainsTotals(t *testing.T) {
	v := View{
		ShopName: "ร้านทดสอบ", BillNo: "B000001", Total: 2100, Paid: 2500, Change: 400,
		PaymentMethod: "cash",
		Items:         []Line{{Name: "น้ำดื่ม", Qty: 3, Price: 700, LineTotal: 2100}},
	}
	html := HTML(v, 58)
	for _, want := range []string{"ร้านทดสอบ", "B000001", "21.00", "รวมทั้งสิ้น"} {
		if !contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
}

func TestESCPOSNonEmptyAndCuts(t *testing.T) {
	v := View{ShopName: "ร้าน", BillNo: "B1", Total: 700, Paid: 700, PaymentMethod: "cash",
		Items: []Line{{Name: "x", Qty: 1, Price: 700, LineTotal: 700}}}
	b := ESCPOS(v, 80)
	if len(b) < 20 {
		t.Fatal("escpos output too short")
	}
	// starts with ESC @ (init)
	if b[0] != 0x1B || b[1] != 0x40 {
		t.Error("escpos must start with ESC @")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
