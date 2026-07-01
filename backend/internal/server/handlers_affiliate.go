package server

import "net/http"

// handleAffiliateRequest: khách hàng gửi yêu cầu trở thành Cộng tác viên bán hàng.
func (s *Server) handleAffiliateRequest(w http.ResponseWriter, r *http.Request) {
	if err := s.affiliate.Request(r.Context(), userID(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"affiliate_status": "pending"})
}

// handleListAffiliateRequests: admin xem các yêu cầu đang chờ duyệt.
func (s *Server) handleListAffiliateRequests(w http.ResponseWriter, r *http.Request) {
	reqs, err := s.affiliate.ListPending(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reqs)
}

func (s *Server) handleApproveAffiliate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.affiliate.Approve(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleRejectAffiliate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.affiliate.Reject(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
