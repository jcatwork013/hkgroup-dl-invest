package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

func parseID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, service.ErrValidation
	}
	return id, nil
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.dashboard.Admin(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleCapTable(w http.ResponseWriter, r *http.Request) {
	rows, err := s.dashboard.CapTable(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleIntegrityCheck(w http.ResponseWriter, r *http.Request) {
	m, err := s.dashboard.IntegrityCheck(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"healthy": len(m) == 0, "mismatches": m})
}

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := int32(100)
	offset := int32(0)
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = int32(v)
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
		offset = int32(v)
	}
	logs, err := s.dashboard.AuditLogs(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	limit := int32(200)
	offset := int32(0)
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = int32(v)
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
		offset = int32(v)
	}
	users, err := s.identity.ListUsers(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user, err := s.identity.AdminCreateUser(r.Context(), userID(r), service.RegisterInput{
		FullName: in.FullName, Phone: in.Phone, Email: in.Email, Password: in.Password,
	}, in.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPublicUser(user))
}

// handleDeleteUser — admin "safe delete": removes the account and ALL its personal data when it has
// no financial footprint; blocks (409) otherwise to protect the shareholder register & audit trail.
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.identity.DeleteUser(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleAdminManualKYC(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user, err := s.identity.AdminSetKYC(r.Context(), userID(r), id, in.Approve, in.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPublicUser(user))
}

// handleAdminUserKYC returns a user's latest KYC submission (ảnh mặt trước/sau + selfie) so the admin
// can review the images before duyệt KYC from the user-management screen. 404 nếu chưa nộp KYC.
func (s *Server) handleAdminUserKYC(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.identity.GetUserKYC(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// handleAdminUserCommissions returns a user's commission wallet (tổng/đã rút/khả dụng) + every
// commission row they earned (as beneficiary), for the read-only admin detail panel. Read-only.
func (s *Server) handleAdminUserCommissions(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	wallet, err := s.wallet.Balance(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	comms, err := s.referral.ListByReferrer(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"wallet": wallet, "commissions": comms})
}

func (s *Server) handleListPendingKYC(w http.ResponseWriter, r *http.Request) {
	recs, err := s.identity.ListPendingKYC(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recs)
}

func (s *Server) handleReviewKYC(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.identity.ReviewKYC(r.Context(), userID(r), id, in.Approve, in.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleAdminListInvestments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	invs, err := s.investment.ListByStatus(r.Context(), db.InvestmentStatus(status))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invs)
}

func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	inv, err := s.investment.Reconcile(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	inv, err := s.investment.ApproveAndIssueShares(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (s *Server) handleRejectInvestment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	_ = decode(r, &in)
	inv, err := s.investment.Reject(r.Context(), userID(r), id, in.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (s *Server) handleApproveCommission(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	c, err := s.referral.ApproveCommission(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handlePayCommission(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	c, err := s.referral.PayCommission(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleListDividends(w http.ResponseWriter, r *http.Request) {
	divs, err := s.dividend.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, divs)
}

func (s *Server) handleListDividendPayouts(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	payouts, err := s.dividend.ListPayouts(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payouts)
}

func (s *Server) handleDeclareDividend(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Period      string `json:"period"`
		TotalAmount int64  `json:"total_amount"`
		Note        string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	div, payouts, err := s.dividend.Declare(r.Context(), userID(r), in.Period, in.TotalAmount, in.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"dividend": div, "payouts": payouts})
}

func (s *Server) handlePayDividend(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := s.dividend.MarkPaid(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleDeleteDividend — admin xoá tay 1 đợt cổ tức (kèm payouts cascade + bản ghi phân bổ liên quan).
func (s *Server) handleDeleteDividend(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.dividend.DeleteDividend(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handlePayAllDividend: admin duyệt 1 lần → chi trả toàn bộ payout của đợt cổ tức (tự động).
func (s *Server) handlePayAllDividend(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	paid, err := s.dividend.PayAll(r.Context(), userID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"paid": paid})
}

// handleSetUserRole: admin đổi vai trò (thăng/giáng chức).
func (s *Server) handleSetUserRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.identity.AdminSetRole(r.Context(), userID(r), id, in.Role); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "role": in.Role})
}

func (s *Server) handleLockUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.identity.AdminLock(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "locked"})
}

func (s *Server) handleUnlockUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.identity.AdminUnlock(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unlocked"})
}

func (s *Server) handleListLockedUsers(w http.ResponseWriter, r *http.Request) {
	ids, err := s.identity.ListLockedUserIDs(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	writeJSON(w, http.StatusOK, map[string]any{"locked": out})
}
