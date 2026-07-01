package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

// AffiliateService: vòng đời khách hàng → CTV. Khách yêu cầu làm affiliate, admin duyệt →
// nâng role saler (mã giới thiệu đã có sẵn từ lúc tạo tài khoản).
type AffiliateService struct {
	store    *store.Store
	identity *IdentityService
}

func NewAffiliateService(s *store.Store, identity *IdentityService) *AffiliateService {
	return &AffiliateService{store: s, identity: identity}
}

// Status trả trạng thái affiliate của user: "affiliate" (đã là CTV), "pending", "rejected", "none".
func (s *AffiliateService) Status(ctx context.Context, u db.User) string {
	if u.Role == db.UserRoleSaler {
		return "affiliate"
	}
	st, err := s.store.GetAffiliateRequestStatus(ctx, u.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "none"
	}
	if err != nil {
		return "none"
	}
	return st // pending | rejected | approved
}

// Request: khách hàng gửi yêu cầu trở thành CTV. Chỉ role=customer mới cần/được yêu cầu.
func (s *AffiliateService) Request(ctx context.Context, userID uuid.UUID) error {
	u, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return ErrNotFound
	}
	if u.Role == db.UserRoleSaler {
		return errors.Join(ErrValidation, errors.New("bạn đã là Cộng tác viên"))
	}
	if u.Role != db.UserRoleCustomer {
		return errors.Join(ErrValidation, errors.New("chỉ tài khoản khách hàng mới đăng ký làm CTV"))
	}
	return s.store.UpsertAffiliateRequest(ctx, userID)
}

func (s *AffiliateService) ListPending(ctx context.Context) ([]db.AffiliateRequestRow, error) {
	return s.store.ListPendingAffiliateRequests(ctx)
}

// Approve: admin duyệt → nâng role lên saler + đánh dấu request approved (trong 1 transaction).
func (s *AffiliateService) Approve(ctx context.Context, admin, target uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.UpdateUserRole(ctx, target, db.UserRoleSaler); e != nil {
			return e
		}
		if e := q.SetAffiliateRequestStatus(ctx, target, "approved", admin); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "affiliate.approve", "users", target.String(), nil, nil)
	})
}

func (s *AffiliateService) Reject(ctx context.Context, admin, target uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.SetAffiliateRequestStatus(ctx, target, "rejected", admin); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "affiliate.reject", "users", target.String(), nil, nil)
	})
}
