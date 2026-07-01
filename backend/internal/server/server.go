package server

import (
	"net/http"
	"time"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/service"
)

// Server wires HTTP handlers to the application services.
type Server struct {
	jwt           *security.JWTManager
	identity      *service.IdentityService
	investment    *service.InvestmentService
	referral      *service.ReferralService
	dividend      *service.DividendService
	dashboard     *service.DashboardService
	settings      *service.SettingsService
	profile       *service.ProfileService
	distribution  *service.DistributionService
	wallet        *service.WalletService
	upload        *service.UploadService
	sales         *service.SalesService
	passwordReset *service.PasswordResetService
	affiliate     *service.AffiliateService
	corsOrigin    string
}

type Deps struct {
	JWT           *security.JWTManager
	Identity      *service.IdentityService
	Investment    *service.InvestmentService
	Referral      *service.ReferralService
	Dividend      *service.DividendService
	Dashboard     *service.DashboardService
	Settings      *service.SettingsService
	Profile       *service.ProfileService
	Distribution  *service.DistributionService
	Wallet        *service.WalletService
	Upload        *service.UploadService
	Sales         *service.SalesService
	PasswordReset *service.PasswordResetService
	Affiliate     *service.AffiliateService
	CORSOrigin    string
}

func New(d Deps) *Server {
	return &Server{
		jwt:           d.JWT,
		identity:      d.Identity,
		investment:    d.Investment,
		referral:      d.Referral,
		dividend:      d.Dividend,
		dashboard:     d.Dashboard,
		settings:      d.Settings,
		profile:       d.Profile,
		distribution:  d.Distribution,
		wallet:        d.Wallet,
		upload:        d.Upload,
		sales:         d.Sales,
		passwordReset: d.PasswordReset,
		affiliate:     d.Affiliate,
		corsOrigin:    d.CORSOrigin,
	}
}

// corsOrigins splits the configured CORS_ORIGIN (comma-separated) into a list, so the API can
// serve both https://duoclieuhk.vn and https://admin.duoclieuhk.vn from one deployment.
func (s *Server) corsOrigins() []string {
	var out []string
	for _, o := range strings.Split(s.corsOrigin, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.corsOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Idempotency-Key"},
		AllowCredentials: true,
	}))

	authRL := newRateLimiter(10, time.Minute) // 10 auth attempts / IP / minute

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// ----- public -----
		r.Group(func(r chi.Router) {
			r.Use(s.rateLimit(authRL))
			r.Post("/auth/register", s.handleRegister)
			r.Post("/auth/register-customer", s.handleRegisterCustomer) // đăng ký KHÁCH HÀNG từ website
			r.Post("/checkout", s.handlePublicCheckout)                 // khách đặt hàng online (giỏ hàng)
			r.Post("/auth/login", s.handleLogin)
			r.Post("/auth/refresh", s.handleRefresh)
			r.Post("/auth/forgot-password", s.handleForgotPassword) // gửi link đặt lại mật khẩu
			r.Post("/auth/reset-password", s.handleResetPassword)   // đặt mật khẩu mới bằng token
		})
		r.Get("/offering", s.handleGetOffering)              // landing page data (no return promise)
		r.Get("/settings", s.handleGetSettings)              // public site settings (contact info, brand year)
		r.Get("/pool", s.handleGetPool)                      // public pool/fundraising status
		r.Get("/public/images/{id}", s.handleGetPublicImage) // ảnh sản phẩm công khai
		r.Get("/products", s.handlePublicProducts)           // catalog công khai cho web bán hàng
		r.Get("/products/{slug}", s.handlePublicProductBySlug)
		r.Get("/policies", s.handlePublicPolicies)
		r.Get("/policies/{slug}", s.handlePublicPolicyBySlug)

		// ----- investor (authenticated) -----
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Get("/me", s.handleMe)
			r.Post("/me/password", s.handleChangePassword) // đổi mật khẩu — dùng chung cho investor & admin
			r.Post("/kyc", s.handleSubmitKYC)
			r.Post("/consent", s.handleConsent)
			r.Post("/me/affiliate-request", s.handleAffiliateRequest) // khách xin làm CTV
			r.Get("/me/orders", s.handleMyCustomerOrders)             // lịch sử đơn MUA của khách (khớp SĐT)
			r.Get("/me/offering", s.handleGetMyOffering) // authed offering (active tiers)
			r.Get("/me/dashboard", s.handleInvestorDashboard)
			r.Get("/me/investments", s.handleMyInvestments)
			r.Get("/me/referrals", s.handleMyReferrals)
			r.Get("/me/dividends", s.handleMyDividends)
			r.Get("/me/profile", s.handleGetMyProfile)
			r.Put("/me/profile", s.handleUpdateMyProfile)
			r.Get("/me/wallet", s.handleGetWallet)
			r.Get("/me/withdrawals", s.handleListMyWithdrawals)
			r.Post("/me/withdrawals", s.handleRequestWithdrawal)
			r.Get("/me/dividend-wallet", s.handleGetDividendWallet)              // số dư cổ tức + lịch rút
			r.Get("/me/dividend-withdrawals", s.handleListMyDividendWithdrawals) // lịch sử rút cổ tức
			r.Post("/me/dividend-withdrawals", s.handleRequestDividendWithdrawal)

			r.Post("/uploads/kyc", s.handleUploadKYC)
			r.Get("/uploads/{id}", s.handleGetUpload)

			r.Post("/investments/contract", s.handleStartContract)
			r.Post("/investments/sign", s.handleSignInvestment)
			r.Post("/investments/{id}/declare-transfer", s.handleDeclareTransfer)
		})

		// ----- admin (authenticated + role=admin) -----
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Use(s.requireRole("admin"))
			r.Get("/admin/dashboard", s.handleAdminDashboard)
			r.Get("/admin/cap-table", s.handleCapTable)
			r.Get("/admin/integrity-check", s.handleIntegrityCheck)
			r.Get("/admin/audit-logs", s.handleAuditLogs)
			r.Get("/admin/settings", s.handleGetAdminSettings)
			r.Put("/admin/settings", s.handleUpdateSettings)
			r.Get("/admin/distributions", s.handleListDistributions)
			r.Post("/admin/distributions", s.handleDistribute)
			r.Delete("/admin/distributions/{id}", s.handleDeleteDistribution)
			r.Post("/admin/distributions/tiered/preview", s.handlePreviewTiered)
			r.Post("/admin/distributions/tiered", s.handleDistributeTiered)
			r.Get("/admin/distributions/sweep/preview", s.handleSweepPreview)
			r.Post("/admin/distributions/sweep", s.handleSweepDividend)
			r.Get("/admin/withdrawals", s.handleListWithdrawals)
			r.Post("/admin/withdrawals/{id}/process", s.handleProcessWithdrawal)
			r.Get("/admin/wallets", s.handleAdminListWallets)
			r.Post("/admin/users/{id}/withdrawals", s.handleAdminCreateWithdrawal)

			r.Get("/admin/kyc/pending", s.handleListPendingKYC)
			r.Post("/admin/kyc/{id}/review", s.handleReviewKYC)

			// User management — admin only (RBAC group). Creating admins is therefore admin-only.
			r.Get("/admin/users", s.handleListUsers)
			r.Post("/admin/users", s.handleAdminCreateUser)
			r.Post("/admin/users/{id}/kyc", s.handleAdminManualKYC)
			r.Get("/admin/users/{id}/kyc", s.handleAdminUserKYC)
			r.Get("/admin/users/{id}/profile", s.handleAdminUserProfile)
			r.Get("/admin/users/{id}/commissions", s.handleAdminUserCommissions)
			r.Delete("/admin/users/{id}", s.handleDeleteUser)
			r.Post("/admin/users/{id}/reset-password", s.handleAdminResetPassword)
			r.Post("/admin/users/{id}/role", s.handleSetUserRole)  // đổi vai trò (thăng/giáng)
			r.Post("/admin/users/{id}/lock", s.handleLockUser)     // khoá tài khoản
			r.Post("/admin/users/{id}/unlock", s.handleUnlockUser) // mở khoá
			r.Get("/admin/locked-users", s.handleListLockedUsers)

			// Funding rounds (vòng gọi vốn) — admin mở vòng mới thủ công khi vòng cũ bán hết.
			r.Get("/admin/offerings", s.handleListOfferings)
			r.Post("/admin/offerings", s.handleOpenRound)

			// Sales catalog — danh mục + sản phẩm (admin).
			// Chính sách (Policy CMS)
			r.Get("/admin/policies", s.handleAdminListPolicies)
			r.Post("/admin/policies", s.handleAdminUpsertPolicy)
			r.Delete("/admin/policies/{slug}", s.handleAdminDeletePolicy)

			r.Get("/admin/categories", s.handleListCategories)
			r.Post("/admin/categories", s.handleCreateCategory)
			r.Put("/admin/categories/{id}", s.handleUpdateCategory)
			r.Delete("/admin/categories/{id}", s.handleDeleteCategory)
			r.Get("/admin/products", s.handleListProducts)
			r.Post("/admin/products", s.handleCreateProduct)
			r.Post("/admin/products/image", s.handleUploadProductImage)
			r.Put("/admin/products/{id}", s.handleUpdateProduct)
			r.Delete("/admin/products/{id}", s.handleDeleteProduct)

			// Bán hàng — admin: toàn bộ đơn + giám sát saler.
			r.Get("/admin/orders", s.handleListAllOrders)
			r.Delete("/admin/orders/{id}", s.handleDeleteOrder)
			r.Get("/admin/salers", s.handleSalerStats)

			// Duyệt yêu cầu làm Cộng tác viên (affiliate).
			r.Get("/admin/affiliate-requests", s.handleListAffiliateRequests)
			r.Post("/admin/affiliate-requests/{id}/approve", s.handleApproveAffiliate)
			r.Post("/admin/affiliate-requests/{id}/reject", s.handleRejectAffiliate)

			// Tier management.
			r.Get("/admin/tiers", s.handleListTiers)
			r.Post("/admin/tiers", s.handleCreateTier)
			r.Put("/admin/tiers/{id}", s.handleUpdateTier)
			r.Post("/admin/tiers/{id}/active", s.handleSetTierActive)
			r.Delete("/admin/tiers/{id}", s.handleDeleteTier)

			r.Get("/admin/investments", s.handleAdminListInvestments)
			r.Post("/admin/investments/{id}/reconcile", s.handleReconcile)
			r.Post("/admin/investments/{id}/approve", s.handleApprove)
			r.Post("/admin/investments/{id}/reject", s.handleRejectInvestment)

			r.Post("/admin/commissions/{id}/approve", s.handleApproveCommission)
			r.Post("/admin/commissions/{id}/pay", s.handlePayCommission)

			r.Get("/admin/dividends", s.handleListDividends)
			r.Post("/admin/dividends", s.handleDeclareDividend)
			r.Delete("/admin/dividends/{id}", s.handleDeleteDividend)
			r.Get("/admin/dividends/{id}/payouts", s.handleListDividendPayouts)
			r.Post("/admin/dividend-payouts/{id}/pay", s.handlePayDividend)
			r.Post("/admin/dividends/{id}/pay-all", s.handlePayAllDividend) // duyệt 1 lần → chi trả tất cả
		})

		// ----- bán hàng: dùng chung admin + saler -----
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Use(s.requireAnyRole("admin", "saler"))
			r.Get("/sales/products", s.handleActiveProducts) // danh sách hàng để tạo đơn
			r.Get("/sales/salers", s.handleListSalers)       // để chọn affiliate
			r.Get("/sales/my-orders", s.handleMyOrders)      // đơn của chính người bán
			r.Get("/sales/my-commissions", s.handleMySalesCommissions)
			r.Post("/sales/orders", s.handleCreateOrder)
			r.Get("/sales/orders/{id}", s.handleOrderDetail)
			r.Post("/sales/orders/{id}/pay", s.handlePayOrder)
			r.Post("/sales/orders/{id}/cancel", s.handleCancelOrder)
		})
	})

	return r
}
