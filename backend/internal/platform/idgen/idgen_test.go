package idgen

import (
	"regexp"
	"testing"
)

func TestInvestmentCodeFormat(t *testing.T) {
	re := regexp.MustCompile(`^HKG-INV-[A-Z2-9]{6}$`)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		c := InvestmentCode()
		if !re.MatchString(c) {
			t.Fatalf("bad code format: %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 990 { // overwhelmingly unique
		t.Fatalf("too many collisions: %d unique of 1000", len(seen))
	}
}

func TestOTPFormat(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	for i := 0; i < 1000; i++ {
		if o := OTP(); !re.MatchString(o) {
			t.Fatalf("bad otp: %q", o)
		}
	}
}
