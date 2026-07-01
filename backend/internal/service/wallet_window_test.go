package service

import (
	"reflect"
	"testing"
	"time"
)

func TestParseWithdrawalDays(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"15,30", []int{15, 30}},
		{" 30 , 15 ", []int{15, 30}},  // trim + sort
		{"15,15,30", []int{15, 30}},   // dedupe
		{"0,15,32,30", []int{15, 30}}, // bỏ ngoài [1,31]
		{"", []int{15, 30}},           // rỗng -> mặc định
		{"abc", []int{15, 30}},        // sai -> mặc định
		{"1,5,31", []int{1, 5, 31}},   // tuỳ ý
	}
	for _, c := range cases {
		if got := parseWithdrawalDays(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseWithdrawalDays(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 12, 0, 0, 0, ictZone)
}

func TestComputeWindow(t *testing.T) {
	days := []int{15, 30}

	// Trước ngày 15 -> mở vào 15 cùng tháng.
	w := computeWindow(days, d(2026, time.July, 10))
	if w.OpenToday || w.NextDate != "2026-07-15" || w.DaysUntil != 5 {
		t.Errorf("10/7: %+v", w)
	}

	// Đúng ngày 15 -> đang mở.
	w = computeWindow(days, d(2026, time.July, 15))
	if !w.OpenToday || w.NextDate != "2026-07-15" || w.DaysUntil != 0 {
		t.Errorf("15/7: %+v", w)
	}

	// Sau 15, trước 30 -> mở vào 30.
	w = computeWindow(days, d(2026, time.July, 20))
	if w.OpenToday || w.NextDate != "2026-07-30" || w.DaysUntil != 10 {
		t.Errorf("20/7: %+v", w)
	}

	// Sau 30 -> mở vào 15 THÁNG SAU.
	w = computeWindow(days, d(2026, time.July, 31))
	if w.OpenToday || w.NextDate != "2026-08-15" {
		t.Errorf("31/7: %+v", w)
	}
}

func TestComputeWindowMonthEndClamp(t *testing.T) {
	days := []int{15, 30}

	// Tháng 2/2026 chỉ có 28 ngày -> "ngày 30" rơi vào NGÀY CUỐI THÁNG (28).
	w := computeWindow(days, d(2026, time.February, 28))
	if !w.OpenToday {
		t.Errorf("28/2 phải mở (clamp ngày 30 -> 28): %+v", w)
	}
	// 27/2 chưa mở; ngày mở kế tiếp là 28/2.
	w = computeWindow(days, d(2026, time.February, 27))
	if w.OpenToday || w.NextDate != "2026-02-28" || w.DaysUntil != 1 {
		t.Errorf("27/2: %+v", w)
	}
}

func TestDaysLabel(t *testing.T) {
	cases := []struct {
		in   []int
		want string
	}{
		{[]int{15, 30}, "ngày 15 và 30"},
		{[]int{15}, "ngày 15"},
		{[]int{15, 30, 31}, "ngày 15, 30 và 31"},
	}
	for _, c := range cases {
		if got := daysLabel(c.in); got != c.want {
			t.Errorf("daysLabel(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
