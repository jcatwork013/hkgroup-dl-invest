package server

import "net/http"

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	// Public: loại bỏ secret (vd resend_api_key) trước khi trả ra.
	settings, err := s.settings.Public(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// GET /api/v1/admin/settings — như public nhưng kèm cờ "<secret>_configured" để admin
// biết đã cấu hình hay chưa; giá trị secret KHÔNG bao giờ trả về.
func (s *Server) handleGetAdminSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settings.Admin(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in map[string]string
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	settings, err := s.settings.Update(r.Context(), userID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}
