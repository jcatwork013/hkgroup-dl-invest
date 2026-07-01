package service

import "errors"

// Domain errors. The HTTP layer maps these to status codes.
var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidCredential = errors.New("invalid credentials")
	ErrConflict          = errors.New("conflict")
	ErrValidation        = errors.New("validation error")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidState      = errors.New("invalid state transition")
	ErrKYCNotApproved    = errors.New("kyc not approved")
	ErrOTPInvalid        = errors.New("otp invalid or expired")
	ErrPoolExhausted     = errors.New("offering pool exhausted")
)
