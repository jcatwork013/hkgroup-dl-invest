package server

import (
	"net/http"

	"github.com/hkgroup/backend/internal/service"
)

// ----- Investor: authenticated offering -----

func (s *Server) handleGetMyOffering(w http.ResponseWriter, r *http.Request) {
	off, tiers, err := s.investment.OfferingForUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"offering": off,
		"tiers":    tiers,
	})
}

// ----- Admin: funding rounds (vòng gọi vốn) -----

func (s *Server) handleListOfferings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.investment.ListOfferings(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleOpenRound(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name          string `json:"name"`
		ValuationVnd  int64  `json:"valuation_vnd"`
		TotalShares   int64  `json:"total_shares"`
		SharesForSale int64  `json:"shares_for_sale"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	off, err := s.investment.OpenNewRound(r.Context(), userID(r), service.OpenNewRoundInput{
		Name:          in.Name,
		ValuationVnd:  in.ValuationVnd,
		TotalShares:   in.TotalShares,
		SharesForSale: in.SharesForSale,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, off)
}

// ----- Admin: tier management -----

func (s *Server) handleListTiers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.investment.ListAllTiers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

type tierBody struct {
	Name         string  `json:"name"`
	AmountVnd    int64   `json:"amount_vnd"`
	Shares       int64   `json:"shares"`
	OwnershipPct float64 `json:"ownership_pct"`
	SortOrder    int32   `json:"sort_order"`
	Active       bool    `json:"active"`
}

func (b tierBody) toInput() service.TierInput {
	return service.TierInput{
		Name:         b.Name,
		AmountVnd:    b.AmountVnd,
		Shares:       b.Shares,
		OwnershipPct: b.OwnershipPct,
		SortOrder:    b.SortOrder,
		Active:       b.Active,
	}
}

func (s *Server) handleCreateTier(w http.ResponseWriter, r *http.Request) {
	var in tierBody
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	t, err := s.investment.CreateTier(r.Context(), userID(r), in.toInput())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleUpdateTier(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in tierBody
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	t, err := s.investment.UpdateTier(r.Context(), userID(r), id, in.toInput())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDeleteTier(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.investment.DeleteTier(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleSetTierActive(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Active bool `json:"active"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	t, err := s.investment.SetTierActive(r.Context(), userID(r), id, in.Active)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
