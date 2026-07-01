package service

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/events"
	"github.com/hkgroup/backend/internal/store"
)

type DividendService struct {
	store  *store.Store
	events events.Publisher
}

func NewDividendService(s *store.Store) *DividendService {
	return &DividendService{store: s, events: events.Noop{}}
}

// SetEvents installs a domain-event publisher (nil-safe: defaults to Noop).
func (s *DividendService) SetEvents(p events.Publisher) {
	if p != nil {
		s.events = p
	}
}

// Declare creates a dividend (INVARIANT 6: exists only because an admin made it — no cron) and
// computes pro-rata payouts across current shareholders. Payouts start UNPAID; admin marks each
// paid when the money actually moves. dividend_payouts.amount is the REAL "money received back".
func (s *DividendService) Declare(ctx context.Context, admin uuid.UUID, period string, total int64, note string) (db.Dividend, []db.DividendPayout, error) {
	if total <= 0 || period == "" {
		return db.Dividend{}, nil, ErrValidation
	}
	var div db.Dividend
	var payouts []db.DividendPayout

	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		holdings, e := q.ListAllShareholdings(ctx)
		if e != nil {
			return e
		}
		var totalShares int64
		for _, h := range holdings {
			totalShares += h.Shares
		}
		if totalShares == 0 {
			return errors.Join(ErrValidation, errors.New("no shareholders to distribute to"))
		}

		div, e = q.CreateDividend(ctx, db.CreateDividendParams{
			DeclaredBy: admin, Period: period, TotalAmount: total, Note: pgText(note),
		})
		if e != nil {
			return e
		}

		for _, h := range holdings {
			amount := int64(math.Floor(float64(total) * float64(h.Shares) / float64(totalShares)))
			payout, e := q.CreateDividendPayout(ctx, db.CreateDividendPayoutParams{
				DividendID: div.ID, UserID: h.UserID, Shares: h.Shares, Amount: amount,
			})
			if errors.Is(e, pgx.ErrNoRows) {
				continue
			}
			if e != nil {
				return e
			}
			payouts = append(payouts, payout)
		}
		return audit.Write(ctx, q, audit.Actor(admin), "dividend.declare", "dividends", div.ID.String(), nil, div)
	})
	if err == nil {
		_ = s.events.Publish(ctx, "hk.dividend.declared", div)
	}
	return div, payouts, err
}

// PayAll chi trả TẤT CẢ payout chưa trả của 1 đợt cổ tức trong MỘT transaction. Admin duyệt 1 lần,
// hệ thống tự tính sẵn & cộng vào ví cổ tức của từng cổ đông — không bấm từng người, tránh sai số.
func (s *DividendService) PayAll(ctx context.Context, admin, dividendID uuid.UUID) (int, error) {
	paid := 0
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		rows, e := q.ListPayoutsByDividend(ctx, dividendID)
		if e != nil {
			return e
		}
		for _, r := range rows {
			if r.PaidAt.Valid {
				continue // đã trả rồi → bỏ qua (idempotent)
			}
			if _, e := q.MarkPayoutPaid(ctx, r.ID); e != nil {
				return e
			}
			paid++
		}
		return audit.Write(ctx, q, audit.Actor(admin), "dividend.pay_all", "dividends", dividendID.String(), nil, map[string]int{"paid": paid})
	})
	return paid, err
}

func (s *DividendService) MarkPaid(ctx context.Context, admin, payoutID uuid.UUID) (db.DividendPayout, error) {
	var p db.DividendPayout
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		p, e = q.MarkPayoutPaid(ctx, payoutID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidState
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "dividend.pay", "dividend_payouts", payoutID.String(), nil, p)
	})
	return p, err
}

func (s *DividendService) List(ctx context.Context) ([]db.Dividend, error) {
	return s.store.ListDividends(ctx)
}

// DeleteDividend xoá tay 1 đợt cổ tức: gỡ các bản ghi revenue_distributions trỏ tới nó (FK NO ACTION)
// TRƯỚC, rồi xoá dividend (dividend_payouts cascade theo FK 00007). Dùng để dọn dữ liệu test/sai.
func (s *DividendService) DeleteDividend(ctx context.Context, admin, dividendID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.DeleteRevenueDistributionsByDividend(ctx, uuid.NullUUID{UUID: dividendID, Valid: true}); e != nil {
			return e
		}
		if e := q.DeleteDividend(ctx, dividendID); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "dividend.delete", "dividends", dividendID.String(), nil, nil)
	})
}

func (s *DividendService) ListPayouts(ctx context.Context, dividendID uuid.UUID) ([]db.ListPayoutsByDividendRow, error) {
	return s.store.ListPayoutsByDividend(ctx, dividendID)
}

func (s *DividendService) ListPayoutsByUser(ctx context.Context, userID uuid.UUID) ([]db.ListPayoutsByUserRow, error) {
	return s.store.ListPayoutsByUser(ctx, userID)
}
