package server

import (
	"net/http"

	"github.com/hkgroup/backend/internal/service"
)

func (s *Server) handleGetMyProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.profile.Get(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	var in service.ProfileInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	p, err := s.profile.Upsert(r.Context(), userID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Admin: view any investor's full profile (kiểm soát thông tin nhà đầu tư).
func (s *Server) handleAdminUserProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := s.profile.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
