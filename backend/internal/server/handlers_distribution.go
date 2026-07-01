package server

import "net/http"

// Public: capital-raise pool status (fundraising progress + distribution totals).
// The invest page shows this section only when settings.show_pool_public = "on".
func (s *Server) handleGetPool(w http.ResponseWriter, r *http.Request) {
	pool, err := s.distribution.Pool(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pool)
}

func (s *Server) handleListDistributions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.distribution.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// handleDeleteDistribution — admin xoá tay 1 lần phân bổ (revenue_distributions + đợt cổ tức gắn kèm).
func (s *Server) handleDeleteDistribution(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.distribution.DeleteDistribution(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDistribute(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Period       string `json:"period"`
		TotalRevenue int64  `json:"total_revenue"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	res, err := s.distribution.Distribute(r.Context(), userID(r), in.Period, in.TotalRevenue)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// handlePreviewTiered computes the 9%+6% "đồng chia + bonus" plan WITHOUT committing — drives the
// admin preview and the investor bonus pie chart.
func (s *Server) handlePreviewTiered(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Period       string `json:"period"`
		TotalRevenue int64  `json:"total_revenue"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	plan, err := s.distribution.PreviewTiered(r.Context(), in.Period, in.TotalRevenue)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// handleSweepPreview đọc số đơn paid chưa gộp + kế hoạch chia (không ghi) — cho nút "Quét cổ tức".
func (s *Server) handleSweepPreview(w http.ResponseWriter, r *http.Request) {
	res, err := s.distribution.SweepPreview(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleSweepDividend gom pool 15% của mọi đơn thành công chưa gộp thành 1 đợt cổ tức thực (backfill
// cả đơn cũ ở lần đầu). Idempotent theo cột swept.
func (s *Server) handleSweepDividend(w http.ResponseWriter, r *http.Request) {
	res, err := s.distribution.SweepDividend(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// handleDistributeTiered commits the 9%+6% plan as a real, audited dividend.
func (s *Server) handleDistributeTiered(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Period       string `json:"period"`
		TotalRevenue int64  `json:"total_revenue"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	res, err := s.distribution.DistributeTiered(r.Context(), userID(r), in.Period, in.TotalRevenue)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}
