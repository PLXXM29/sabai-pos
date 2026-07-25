package handler

import "testing"

func TestParseBahtToSatang(t *testing.T) {
	cases := []struct {
		text string
		want int64
		ok   bool
	}{
		{"รับเงิน 7.00 บาท", 700, true},
		{"เงินเข้า 1,234.56 บาท ยอดคงเหลือ 9,999.00", 123456, true},
		{"+12.00 จากพร้อมเพย์", 1200, true},
		{"received 250.75 THB", 25075, true},
		{"คงเหลือ 500.00 บาท รับโอน 42.50 บาท", 4250, true}, // prefer the received amount
		{"สวัสดีครับ", 0, false},
		{"โอน 0.00", 0, false},
	}
	for _, c := range cases {
		got, ok := parseBahtToSatang(c.text, nil)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parse(%q) = (%d,%v), want (%d,%v)", c.text, got, ok, c.want, c.ok)
		}
	}
}
