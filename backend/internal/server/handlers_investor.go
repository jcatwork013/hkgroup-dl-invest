package server

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/service"
)

func (s *Server) handleGetOffering(w http.ResponseWriter, r *http.Request) {
	off, tiers, err := s.investment.Offering(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	// Landing payload: valuation, pool progress, tiers (shares & %), NO return/profit figure.
	writeJSON(w, http.StatusOK, map[string]any{
		"offering": off,
		"tiers":    tiers,
	})
}

// handleMe returns the current authenticated user (fresh KYC status + message for the bell).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.identity.GetUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPublicUser(u))
}

func (s *Server) handleSubmitKYC(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CCCDNumber   string `json:"cccd_number"`
		CCCDImageURL string `json:"cccd_image_url"`
		CCCDBackURL  string `json:"cccd_back_url"`
		SelfieURL    string `json:"selfie_url"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.identity.SubmitKYC(r.Context(), userID(r), in.CCCDNumber, in.CCCDImageURL, in.CCCDBackURL, in.SelfieURL)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Type string `json:"type"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.identity.RecordConsent(r.Context(), userID(r), in.Type, clientIP(r), r.UserAgent()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

func (s *Server) handleStartContract(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TierID string `json:"tier_id"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	tierID, err := uuid.Parse(in.TierID)
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	res, err := s.investment.StartContract(r.Context(), userID(r), tierID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleSignInvestment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ContractID string `json:"contract_id"`
		OTPRef     string `json:"otp_ref"`
		OTPCode    string `json:"otp_code"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	contractID, err := uuid.Parse(in.ContractID)
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	res, err := s.investment.SignAndCreateInvestment(r.Context(), service.SignInput{
		UserID:         userID(r),
		ContractID:     contractID,
		OTPRef:         in.OTPRef,
		OTPCode:        in.OTPCode,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleDeclareTransfer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	pay, err := s.investment.DeclareTransfer(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pay)
}

func (s *Server) handleInvestorDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.dashboard.Investor(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleMyInvestments(w http.ResponseWriter, r *http.Request) {
	invs, err := s.investment.ListByUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invs)
}

func (s *Server) handleMyReferrals(w http.ResponseWriter, r *http.Request) {
	refs, err := s.referral.ListReferralsByReferrer(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	comms, err := s.referral.ListByReferrer(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"referrals": refs, "commissions": comms})
}

func (s *Server) handleMyDividends(w http.ResponseWriter, r *http.Request) {
	payouts, err := s.dividend.ListPayoutsByUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payouts)
}
