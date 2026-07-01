package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/otp"
	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/service"
	"github.com/hkgroup/backend/internal/store"
)

// These are integration tests. They require a migrated Postgres and a Redis instance.
// Set TEST_DATABASE_URL and TEST_REDIS_URL; otherwise the suite is skipped.
//
//	TEST_DATABASE_URL=postgres://hk:hk_dev_password@localhost:55432/hkgroup?sslmode=disable \
//	TEST_REDIS_URL=redis://localhost:56379/0 go test ./internal/service/...

type testEnv struct {
	t          *testing.T
	store      *store.Store
	identity   *service.IdentityService
	investment *service.InvestmentService
	referral   *service.ReferralService
	dividend   *service.DividendService
	dashboard  *service.DashboardService
	admin      db.User
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	redisURL := os.Getenv("TEST_REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Skip("set TEST_DATABASE_URL and TEST_REDIS_URL to run integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pg connect: %v", err)
	}
	t.Cleanup(pool.Close)

	st := store.New(pool)
	truncate(t, pool)

	ropt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(ropt)
	t.Cleanup(func() { _ = rdb.Close() })

	jwt := security.NewJWTManager("test-secret-test-secret-test-sec", 15*time.Minute, 24*time.Hour)
	otpSvc := otp.New(rdb)
	ref := service.NewReferralService(st, 0.05, 0.10, true)
	ident := service.NewIdentityService(st, jwt, ref)
	inv := service.NewInvestmentService(st, otpSvc, ref, service.CompanyBank{
		Bank: "Vietcombank", Account: "0123456789", AccountName: "CONG TY CO PHAN HKGROUP",
	})
	div := service.NewDividendService(st)
	dash := service.NewDashboardService(st)

	hash, _ := security.HashPassword("Admin@12345")
	admin, err := st.EnsureAdmin(ctx, db.EnsureAdminParams{
		FullName: "Admin", Phone: "0900000000", Email: "admin@test.vn",
		PasswordHash: hash, ReferralCode: "ADMIN1",
	})
	if err != nil {
		t.Fatalf("ensure admin: %v", err)
	}

	return &testEnv{t: t, store: st, identity: ident, investment: inv, referral: ref, dividend: div, dashboard: dash, admin: admin}
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		TRUNCATE users, kyc_records, consents, contracts, investments, payments,
		         shareholdings, share_ledger, referrals, commissions,
		         dividends, dividend_payouts, audit_logs RESTART IDENTITY CASCADE;
		UPDATE offering SET shares_sold = 0;`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// ----- shared flow helpers -----

func (e *testEnv) ctx() context.Context { return context.Background() }

// newApprovedInvestor registers a user and gets their eKYC approved.
func (e *testEnv) newApprovedInvestor(email string, opts ...func(*service.RegisterInput)) db.User {
	e.t.Helper()
	in := service.RegisterInput{FullName: "NĐT " + email, Phone: phoneFor(email), Email: email, Password: "Password123"}
	for _, o := range opts {
		o(&in)
	}
	user, _, err := e.identity.Register(e.ctx(), in)
	if err != nil {
		e.t.Fatalf("register %s: %v", email, err)
	}
	rec, err := e.identity.SubmitKYC(e.ctx(), user.ID, "012345678901", "s3://cccd", "s3://cccd-back", "s3://selfie")
	if err != nil {
		e.t.Fatalf("submit kyc: %v", err)
	}
	if _, err := e.identity.ReviewKYC(e.ctx(), e.admin.ID, rec.ID, true, ""); err != nil {
		e.t.Fatalf("approve kyc: %v", err)
	}
	return user
}

func (e *testEnv) activeOffering() db.Offering {
	e.t.Helper()
	off, err := e.store.GetActiveOffering(e.ctx())
	if err != nil {
		e.t.Fatalf("offering: %v", err)
	}
	return off
}

// createPendingInvestment runs contract->sign->declare and returns the pending investment.
func (e *testEnv) createPendingInvestment(user db.User, tier db.InvestmentTier) db.Investment {
	e.t.Helper()
	start, err := e.investment.StartContract(e.ctx(), user.ID, tier.ID)
	if err != nil {
		e.t.Fatalf("start contract: %v", err)
	}
	created, err := e.investment.SignAndCreateInvestment(e.ctx(), service.SignInput{
		UserID: user.ID, ContractID: start.Contract.ID, OTPRef: start.OTPRef, OTPCode: start.OTPCode,
		IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		e.t.Fatalf("sign: %v", err)
	}
	if _, err := e.investment.DeclareTransfer(e.ctx(), user.ID, created.Investment.ID); err != nil {
		e.t.Fatalf("declare: %v", err)
	}
	return created.Investment
}

// fullInvest runs the whole flow through approval+issuance and returns the approved investment.
func (e *testEnv) fullInvest(user db.User, tier db.InvestmentTier) db.Investment {
	e.t.Helper()
	inv := e.createPendingInvestment(user, tier)
	if _, err := e.investment.Reconcile(e.ctx(), e.admin.ID, inv.ID); err != nil {
		e.t.Fatalf("reconcile: %v", err)
	}
	approved, err := e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID)
	if err != nil {
		e.t.Fatalf("approve: %v", err)
	}
	return approved
}

func phoneFor(email string) string {
	// deterministic, unique-enough phone per email for the UNIQUE constraint
	h := uint32(2166136261)
	for _, c := range email {
		h = (h ^ uint32(c)) * 16777619
	}
	return "09" + padLeft(h%100000000, 8)
}

func padLeft(n uint32, width int) string {
	s := itoa(int(n))
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
