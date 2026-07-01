package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

// ictZone — múi giờ Việt Nam (UTC+7). Ngày rút tính theo giờ VN, mở cửa 00h00.
var ictZone = time.FixedZone("ICT", 7*3600)

// WalletService: referral-commission wallet + withdrawal requests (admin-processed).
type WalletService struct {
	store    *store.Store
	settings *SettingsService
}

func NewWalletService(s *store.Store, settings *SettingsService) *WalletService {
	return &WalletService{store: s, settings: settings}
}

type Wallet struct {
	Earned    int64 `json:"earned_vnd"`    // tổng hoa hồng (net) — mọi trạng thái trừ rejected (gồm cả chưa duyệt)
	Withdrawn int64 `json:"withdrawn_vnd"` // đã rút + đang chờ (trừ rejected)
	Available int64 `json:"available_vnd"` // số dư khả dụng = earned - withdrawn
	Pending   int64 `json:"pending_vnd"`   // hoa hồng CHƯA DUYỆT (đã gộp trong earned/available) — chờ admin duyệt
}

// WithdrawalWindow — lịch rút tiền: những ngày trong tháng được phép gửi yêu cầu rút.
type WithdrawalWindow struct {
	Days      []int  `json:"days"`       // các ngày rút (vd [15, 30])
	Today     int    `json:"today"`      // ngày hiện tại (giờ VN)
	OpenToday bool   `json:"open_today"` // hôm nay có mở cửa rút không
	NextDate  string `json:"next_date"`  // ngày mở cửa gần nhất (>= hôm nay), dạng YYYY-MM-DD
	DaysUntil int    `json:"days_until"` // số ngày còn lại tới NextDate (0 nếu đang mở)
}

// parseWithdrawalDays đọc CSV "15,30" -> []int đã chuẩn hoá (1..31, unique, sorted).
// Rỗng/sai -> mặc định [15, 30].
func parseWithdrawalDays(s string) []int {
	seen := map[int]bool{}
	out := make([]int, 0, 4)
	for _, part := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 || n > 31 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Ints(out)
	if len(out) == 0 {
		return []int{15, 30}
	}
	return out
}

func lastDayOfMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).AddDate(0, 1, -1).Day()
}

// isWithdrawalDay: d có phải ngày rút không. Biên tháng: nếu ngày cấu hình vượt số ngày
// của tháng (vd 30/2) thì NGÀY CUỐI THÁNG được tính là ngày rút thay thế.
func isWithdrawalDay(d time.Time, days []int) bool {
	last := lastDayOfMonth(d)
	dd := d.Day()
	for _, t := range days {
		if t == dd || (t > last && dd == last) {
			return true
		}
	}
	return false
}

// computeWindow tính cửa sổ rút dựa trên danh sách ngày + thời điểm tham chiếu (giờ VN).
func computeWindow(days []int, now time.Time) WithdrawalWindow {
	w := WithdrawalWindow{Days: days, Today: now.Day()}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := 0; i < 63; i++ { // quét tối đa ~2 tháng để chắc chắn gặp ngày mở kế tiếp
		d := start.AddDate(0, 0, i)
		if isWithdrawalDay(d, days) {
			w.OpenToday = i == 0
			w.NextDate = d.Format("2006-01-02")
			w.DaysUntil = i
			return w
		}
	}
	return w
}

// daysLabel: [15,30] -> "ngày 15 và 30"; [15,30,31] -> "ngày 15, 30 và 31".
func daysLabel(days []int) string {
	if len(days) == 0 {
		return "ngày rút cố định"
	}
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	if len(parts) == 1 {
		return "ngày " + parts[0]
	}
	return "ngày " + strings.Join(parts[:len(parts)-1], ", ") + " và " + parts[len(parts)-1]
}

// Window trả lịch rút hiện tại theo cấu hình admin (key withdrawal_days) và giờ VN.
func (s *WalletService) Window(ctx context.Context) WithdrawalWindow {
	raw := s.settings.Str(ctx, "withdrawal_days", "15,30")
	return computeWindow(parseWithdrawalDays(raw), time.Now().In(ictZone))
}

func (s *WalletService) Balance(ctx context.Context, userID uuid.UUID) (Wallet, error) {
	earned, err := s.store.SumCommissionEarnedByReferrer(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	// Ví CHUNG: cộng cả hoa hồng bán hàng (seller/affiliate) — consistent với hoa hồng đầu tư.
	salesEarned, err := s.store.SumSalesCommissionEarnedByBeneficiary(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	earned += salesEarned
	withdrawn, err := s.store.SumWithdrawalsByUser(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	// Phần CHƯA DUYỆT (hoa hồng đầu tư + bán hàng) — để nhắc admin duyệt; đã nằm trong earned/available.
	pending, err := s.store.SumPendingCommissionByBeneficiary(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	salesPending, err := s.store.SumPendingSalesCommissionByBeneficiary(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	pending += salesPending
	return Wallet{Earned: earned, Withdrawn: withdrawn, Available: earned - withdrawn, Pending: pending}, nil
}

// RequestWithdrawal creates a pending withdrawal if the wallet has enough available balance.
func (s *WalletService) RequestWithdrawal(ctx context.Context, userID uuid.UUID, amount int64, note string) (db.Withdrawal, error) {
	if amount <= 0 {
		return db.Withdrawal{}, ErrValidation
	}
	// Chỉ cho gửi yêu cầu rút vào ĐÚNG ngày rút (mặc định 15 & 30, mở cửa 00h00 giờ VN).
	win := s.Window(ctx)
	if !win.OpenToday {
		return db.Withdrawal{}, errors.Join(ErrValidation,
			fmt.Errorf("chỉ được gửi yêu cầu rút vào %s hàng tháng — kỳ rút kế tiếp: %s", daysLabel(win.Days), win.NextDate))
	}
	bal, err := s.Balance(ctx, userID)
	if err != nil {
		return db.Withdrawal{}, err
	}
	if amount > bal.Available {
		return db.Withdrawal{}, errors.Join(ErrValidation, errors.New("số dư khả dụng không đủ"))
	}
	return s.store.CreateWithdrawal(ctx, db.CreateWithdrawalParams{UserID: userID, Amount: amount, Note: note, Source: "commission"})
}

func (s *WalletService) ListMine(ctx context.Context, userID uuid.UUID) ([]db.Withdrawal, error) {
	return s.store.ListWithdrawalsByUser(ctx, userID)
}

// ----- VÍ CỔ TỨC (dividend) — số dư rút riêng, cùng lịch rút với ví hoa hồng -----

// DividendBalance: số dư cổ tức = tổng cổ tức được chia − đã rút/đang chờ (source='dividend').
// Pending = 0 vì cổ tức do admin chia (không có khái niệm "chưa duyệt" như hoa hồng).
func (s *WalletService) DividendBalance(ctx context.Context, userID uuid.UUID) (Wallet, error) {
	earned, err := s.store.SumDividendPayoutsByUser(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	withdrawn, err := s.store.SumDividendWithdrawalsByUser(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{Earned: earned, Withdrawn: withdrawn, Available: earned - withdrawn, Pending: 0}, nil
}

// RequestDividendWithdrawal — nhà đầu tư tự rút từ ví CỔ TỨC. Cùng ràng buộc ngày rút
// và kiểm tra số dư như ví hoa hồng, nhưng tính trên số dư cổ tức và ghi source='dividend'.
func (s *WalletService) RequestDividendWithdrawal(ctx context.Context, userID uuid.UUID, amount int64, note string) (db.Withdrawal, error) {
	if amount <= 0 {
		return db.Withdrawal{}, ErrValidation
	}
	win := s.Window(ctx)
	if !win.OpenToday {
		return db.Withdrawal{}, errors.Join(ErrValidation,
			fmt.Errorf("chỉ được gửi yêu cầu rút vào %s hàng tháng — kỳ rút kế tiếp: %s", daysLabel(win.Days), win.NextDate))
	}
	bal, err := s.DividendBalance(ctx, userID)
	if err != nil {
		return db.Withdrawal{}, err
	}
	if amount > bal.Available {
		return db.Withdrawal{}, errors.Join(ErrValidation, errors.New("số dư cổ tức khả dụng không đủ"))
	}
	return s.store.CreateWithdrawal(ctx, db.CreateWithdrawalParams{UserID: userID, Amount: amount, Note: note, Source: "dividend"})
}

func (s *WalletService) ListMyDividendWithdrawals(ctx context.Context, userID uuid.UUID) ([]db.Withdrawal, error) {
	return s.store.ListDividendWithdrawalsByUser(ctx, userID)
}

// AdminListWallets — số dư ví hoa hồng của mọi tài khoản có phát sinh hoa hồng,
// để admin xem ai còn rút được và "lập lệnh rút dùm".
func (s *WalletService) AdminListWallets(ctx context.Context) ([]db.ListWalletBalancesRow, error) {
	return s.store.ListWalletBalances(ctx)
}

// AdminRequestWithdrawal — admin lập lệnh rút DÙM cho 1 user. Khác bản tự rút:
// KHÔNG ràng buộc ngày rút (admin chủ động bất kỳ lúc nào) nhưng VẪN kiểm tra số
// dư khả dụng và ghi audit với actor = admin.
func (s *WalletService) AdminRequestWithdrawal(ctx context.Context, admin, target uuid.UUID, amount int64, note string) (db.Withdrawal, error) {
	if amount <= 0 {
		return db.Withdrawal{}, ErrValidation
	}
	bal, err := s.Balance(ctx, target)
	if err != nil {
		return db.Withdrawal{}, err
	}
	if amount > bal.Available {
		return db.Withdrawal{}, errors.Join(ErrValidation, errors.New("số dư khả dụng không đủ"))
	}
	var wd db.Withdrawal
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		wd, e = q.CreateWithdrawal(ctx, db.CreateWithdrawalParams{UserID: target, Amount: amount, Note: note, Source: "commission"})
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "withdrawal.admin_create", "withdrawals", wd.ID.String(), nil, wd)
	})
	return wd, err
}

func (s *WalletService) ListAll(ctx context.Context) ([]db.ListWithdrawalsRow, error) {
	return s.store.ListWithdrawals(ctx)
}

// Process sets a withdrawal status (approved | paid | rejected) — admin only.
func (s *WalletService) Process(ctx context.Context, admin, id uuid.UUID, status string) (db.Withdrawal, error) {
	st := db.WithdrawalStatus(status)
	switch st {
	case db.WithdrawalStatusApproved, db.WithdrawalStatusPaid, db.WithdrawalStatusRejected:
	default:
		return db.Withdrawal{}, ErrValidation
	}
	var w db.Withdrawal
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		w, e = q.SetWithdrawalStatus(ctx, db.SetWithdrawalStatusParams{
			ID: id, Status: st, ProcessedBy: uuid.NullUUID{UUID: admin, Valid: true},
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "withdrawal."+status, "withdrawals", id.String(), nil, w)
	})
	return w, err
}
