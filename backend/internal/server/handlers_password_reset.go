package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/service"
)

// resetBaseURL — URL gốc của web để dựng link đặt lại mật khẩu. Ưu tiên cấu hình
// app_base_url (admin đặt, hữu ích khi chạy sau proxy), sau đó tới Origin của request.
func (s *Server) resetBaseURL(r *http.Request) string {
	if u := strings.TrimSpace(s.settings.Str(r.Context(), "app_base_url", "")); u != "" {
		return u
	}
	if o := strings.TrimSpace(r.Header.Get("Origin")); o != "" {
		return o
	}
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

// POST /api/v1/auth/forgot-password — self-service: gửi link đặt lại tới email.
// Luôn trả 200 khi email không tồn tại (chống dò email); chỉ báo lỗi khi tính năng
// chưa cấu hình hoặc gửi email thất bại.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.passwordReset.RequestReset(r.Context(), in.Email, s.resetBaseURL(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/v1/auth/reset-password — đặt mật khẩu mới bằng token từ link email.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.passwordReset.ResetPassword(r.Context(), in.Token, in.NewPassword); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/v1/admin/users/{id}/reset-password — admin gửi link đặt lại cho 1 tài khoản.
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	if err := s.passwordReset.AdminRequestReset(r.Context(), userID(r), id, s.resetBaseURL(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
