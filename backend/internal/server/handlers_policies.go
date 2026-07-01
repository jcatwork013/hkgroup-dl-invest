package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

// ---- Công khai (web bán hàng đọc chính sách) ----

func (s *Server) handlePublicPolicies(w http.ResponseWriter, r *http.Request) {
	items, err := s.settings.ListActivePolicies(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policies": items})
}

func (s *Server) handlePublicPolicyBySlug(w http.ResponseWriter, r *http.Request) {
	p, err := s.settings.GetPolicy(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---- Admin (CRUD) ----

func (s *Server) handleAdminListPolicies(w http.ResponseWriter, r *http.Request) {
	items, err := s.settings.ListPolicies(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleAdminUpsertPolicy(w http.ResponseWriter, r *http.Request) {
	var in db.UpsertPolicyParams
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	p, err := s.settings.UpsertPolicy(r.Context(), userID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleAdminDeletePolicy(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, service.ErrValidation)
		return
	}
	if err := s.settings.DeletePolicy(r.Context(), userID(r), slug); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
