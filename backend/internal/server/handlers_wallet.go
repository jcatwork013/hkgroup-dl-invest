package server

import (
	"net/http"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

func (s *Server) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	bal, err := s.wallet.Balance(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		service.Wallet
		Window service.WithdrawalWindow `json:"window"`
	}{Wallet: bal, Window: s.wallet.Window(r.Context())})
}

func (s *Server) handleRequestWithdrawal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	wd, err := s.wallet.RequestWithdrawal(r.Context(), userID(r), in.Amount, in.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wd)
}

func (s *Server) handleListMyWithdrawals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.wallet.ListMine(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// ----- VÍ CỔ TỨC (dividend) — số dư rút riêng, cùng lịch rút với ví hoa hồng -----

func (s *Server) handleGetDividendWallet(w http.ResponseWriter, r *http.Request) {
	bal, err := s.wallet.DividendBalance(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		service.Wallet
		Window service.WithdrawalWindow `json:"window"`
	}{Wallet: bal, Window: s.wallet.Window(r.Context())})
}

func (s *Server) handleRequestDividendWithdrawal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	wd, err := s.wallet.RequestDividendWithdrawal(r.Context(), userID(r), in.Amount, in.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wd)
}

func (s *Server) handleListMyDividendWithdrawals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.wallet.ListMyDividendWithdrawals(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleListWithdrawals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.wallet.ListAll(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if rows == nil {
		rows = []db.ListWithdrawalsRow{}
	}
	writeJSON(w, http.StatusOK, struct {
		Items  []db.ListWithdrawalsRow  `json:"items"`
		Window service.WithdrawalWindow `json:"window"`
	}{Items: rows, Window: s.wallet.Window(r.Context())})
}

// handleAdminListWallets — danh sách số dư ví hoa hồng mọi tài khoản (cho admin rút dùm).
func (s *Server) handleAdminListWallets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.wallet.AdminListWallets(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if rows == nil {
		rows = []db.ListWalletBalancesRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// handleAdminCreateWithdrawal — admin lập lệnh rút DÙM cho user {id} (bỏ qua ràng buộc ngày).
func (s *Server) handleAdminCreateWithdrawal(w http.ResponseWriter, r *http.Request) {
	target, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	wd, err := s.wallet.AdminRequestWithdrawal(r.Context(), userID(r), target, in.Amount, in.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wd)
}

func (s *Server) handleProcessWithdrawal(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	wd, err := s.wallet.Process(r.Context(), userID(r), id, in.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wd)
}
