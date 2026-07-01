package idgen

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars

// InvestmentCode returns a human-readable, unique-enough investment code: HKG-INV-XXXXXX.
func InvestmentCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out := make([]byte, 6)
	for i, x := range b {
		out[i] = codeAlphabet[int(x)%len(codeAlphabet)]
	}
	return "HKG-INV-" + string(out)
}

// SalesOrderCode returns a human-readable, unique-enough sales order code: HKG-SO-XXXXXX.
func SalesOrderCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out := make([]byte, 6)
	for i, x := range b {
		out[i] = codeAlphabet[int(x)%len(codeAlphabet)]
	}
	return "HKG-SO-" + string(out)
}

// ReferralCode returns a short public referral handle.
func ReferralCode() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	out := make([]byte, 5)
	for i, x := range b {
		out[i] = codeAlphabet[int(x)%len(codeAlphabet)]
	}
	return "HK" + string(out)
}

// OTP returns a 6-digit numeric one-time code.
func OTP() string {
	var v [8]byte
	_, _ = rand.Read(v[:])
	n := binary.BigEndian.Uint64(v[:]) % 1000000
	return fmt.Sprintf("%06d", n)
}
