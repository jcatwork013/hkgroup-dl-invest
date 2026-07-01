package server

import (
	"io"
	"net/http"

	"github.com/hkgroup/backend/internal/service"
)

// POST /api/v1/uploads/kyc — investor uploads a confidential KYC image (encrypted at rest).
// multipart field "file"; query/form "kind" = cccd | cccd_back | selfie.
func (s *Server) handleUploadKYC(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	defer file.Close()

	kind := r.FormValue("kind")
	if kind != "cccd" && kind != "cccd_back" && kind != "selfie" {
		kind = "cccd"
	}
	data, err := io.ReadAll(io.LimitReader(file, 13<<20))
	if err != nil {
		writeError(w, err)
		return
	}
	contentType := hdr.Header.Get("Content-Type")

	up, err := s.upload.Save(r.Context(), userID(r), kind, contentType, data)
	if err != nil {
		writeError(w, err)
		return
	}
	// Return an app URL the client stores on the KYC record.
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":  up.ID.String(),
		"url": "/api/v1/uploads/" + up.ID.String(),
	})
}

// GET /api/v1/uploads/{id} — stream the decrypted image. Access: owner or admin.
func (s *Server) handleGetUpload(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	data, contentType, err := s.upload.Load(r.Context(), id, userID(r), userRole(r) == "admin")
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
