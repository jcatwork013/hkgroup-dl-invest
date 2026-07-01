package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hkgroup/backend/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type errBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// writeError maps domain errors to HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errBody{err.Error(), "not_found"})
	case errors.Is(err, service.ErrInvalidCredential):
		writeJSON(w, http.StatusUnauthorized, errBody{"invalid credentials", "unauthorized"})
	case errors.Is(err, service.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errBody{"forbidden", "forbidden"})
	case errors.Is(err, service.ErrConflict):
		writeJSON(w, http.StatusConflict, errBody{err.Error(), "conflict"})
	case errors.Is(err, service.ErrKYCNotApproved):
		writeJSON(w, http.StatusForbidden, errBody{"kyc not approved", "kyc_required"})
	case errors.Is(err, service.ErrOTPInvalid):
		writeJSON(w, http.StatusBadRequest, errBody{"otp invalid or expired", "otp_invalid"})
	case errors.Is(err, service.ErrInvalidState):
		writeJSON(w, http.StatusConflict, errBody{err.Error(), "invalid_state"})
	case errors.Is(err, service.ErrPoolExhausted):
		writeJSON(w, http.StatusConflict, errBody{"offering pool exhausted", "pool_exhausted"})
	case errors.Is(err, service.ErrValidation):
		writeJSON(w, http.StatusBadRequest, errBody{err.Error(), "validation"})
	default:
		writeJSON(w, http.StatusInternalServerError, errBody{"internal error", "internal"})
	}
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.Join(service.ErrValidation, err)
	}
	return nil
}
