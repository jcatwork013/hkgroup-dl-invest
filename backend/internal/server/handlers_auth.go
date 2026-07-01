package server

import (
	"net/http"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

type publicUser struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Role         string `json:"role"`
	KYCStatus    string `json:"kyc_status"`
	KYCMessage   string `json:"kyc_message"`
	ReferralCode string `json:"referral_code"`
}

func toPublicUser(u db.User) publicUser {
	return publicUser{
		ID: u.ID.String(), FullName: u.FullName, Email: u.Email, Phone: u.Phone,
		Role: string(u.Role), KYCStatus: string(u.KycStatus), KYCMessage: u.KycMessage,
		ReferralCode: u.ReferralCode,
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FullName     string `json:"full_name"`
		Phone        string `json:"phone"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		ReferralCode string `json:"referral_code"`
		ReferralType string `json:"referral_type"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user, tokens, err := s.identity.Register(r.Context(), service.RegisterInput{
		FullName: in.FullName, Phone: in.Phone, Email: in.Email, Password: in.Password,
		ReferralCode: in.ReferralCode, ReferralType: in.ReferralType,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": toPublicUser(user), "tokens": tokens})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user, tokens, err := s.identity.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": toPublicUser(user), "tokens": tokens})
}

// POST /api/v1/me/password — authenticated user (investor or admin) changes their own password.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.identity.ChangePassword(r.Context(), userID(r), in.CurrentPassword, in.NewPassword); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	tokens, err := s.identity.Refresh(r.Context(), in.RefreshToken)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}
