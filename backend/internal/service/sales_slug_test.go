package service

import "testing"

func TestSlugifyVN(t *testing.T) {
	cases := map[string]string{
		"Trà Nhuận Tràng":       "tra-nhuan-trang",
		"BỘT CACAO 3IN1 500g":   "bot-cacao-3in1-500g",
		"Đông Trùng Hạ Thảo":    "dong-trung-ha-thao",
		"  Nhân Sâm  Lên Men  ": "nhan-sam-len-men",
		"Trà #1 @Shop!":         "tra-1-shop",
		"---":                   "sp", // không còn ký tự → fallback
		"":                      "sp",
	}
	for in, want := range cases {
		if got := slugifyVN(in); got != want {
			t.Errorf("slugifyVN(%q) = %q, want %q", in, got, want)
		}
	}
}
