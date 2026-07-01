package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hkgroup/backend/internal/config"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/events"
	"github.com/hkgroup/backend/internal/platform/idgen"
	"github.com/hkgroup/backend/internal/platform/otp"
	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/server"
	"github.com/hkgroup/backend/internal/service"
	"github.com/hkgroup/backend/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// ----- Postgres -----
	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	st := store.New(pool)
	log.Info("connected to postgres")

	// ----- Redis (OTP / cache) -----
	ropt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return err
	}
	rdb := redis.NewClient(ropt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return err
	}
	defer rdb.Close()
	log.Info("connected to redis")

	// ----- NATS JetStream (optional; non-fatal) -----
	var publisher events.Publisher = events.Noop{}
	if nps, err := events.ConnectNATS(cfg.NATSURL); err != nil {
		log.Warn("nats unavailable, events disabled", "err", err)
	} else {
		publisher = nps
		defer nps.Close()
		log.Info("connected to nats jetstream")
	}

	// ----- Security & services -----
	jwt := security.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	otpSvc := otp.New(rdb)

	referralSvc := service.NewReferralService(st, cfg.CustomerCommissionRate, cfg.PITRate, cfg.InvestorReferralCashEnabled)
	identitySvc := service.NewIdentityService(st, jwt, referralSvc)
	investmentSvc := service.NewInvestmentService(st, otpSvc, referralSvc, service.CompanyBank{
		Bank: cfg.CompanyBank, Account: cfg.CompanyAccount, AccountName: cfg.CompanyAccountName,
	})
	investmentSvc.SetEvents(publisher)
	dividendSvc := service.NewDividendService(st)
	dividendSvc.SetEvents(publisher)
	dashboardSvc := service.NewDashboardService(st)
	settingsSvc := service.NewSettingsService(st)
	profileSvc := service.NewProfileService(st)
	distributionSvc := service.NewDistributionService(st, dividendSvc, settingsSvc)
	walletSvc := service.NewWalletService(st, settingsSvc)
	salesSvc := service.NewSalesService(st, settingsSvc)
	passwordResetSvc := service.NewPasswordResetService(st, settingsSvc)
	cryptor, err := security.NewCryptor(cfg.KYCEncKey)
	if err != nil {
		return err
	}
	uploadSvc, err := service.NewUploadService(st, cryptor, cfg.UploadDir)
	if err != nil {
		return err
	}
	referralSvc.SetSettings(settingsSvc)   // enable F1 (1-level) investor referral commission
	investmentSvc.SetSettings(settingsSvc) // payment uses the admin-configured company account

	// ----- Bootstrap admin (idempotent) -----
	if err := ensureAdmin(ctx, st, cfg); err != nil {
		return err
	}

	srv := server.New(server.Deps{
		JWT:          jwt,
		Identity:     identitySvc,
		Investment:   investmentSvc,
		Referral:     referralSvc,
		Dividend:     dividendSvc,
		Dashboard:    dashboardSvc,
		Settings:     settingsSvc,
		Profile:      profileSvc,
		Distribution: distributionSvc,
		Wallet:        walletSvc,
		Upload:        uploadSvc,
		Sales:         salesSvc,
		PasswordReset: passwordResetSvc,
		CORSOrigin:    cfg.CORSOrigin,
	})

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// ----- Graceful shutdown -----
	go func() {
		log.Info("http server listening", "port", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}

// ensureAdmin creates (or promotes) the bootstrap admin account. HARD CONSTRAINT-safe: this is
// the only auto-created account and it never touches money/shares.
func ensureAdmin(ctx context.Context, st *store.Store, cfg config.Config) error {
	hash, err := security.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	_, err = st.EnsureAdmin(ctx, db.EnsureAdminParams{
		FullName:     "HKGroup Admin",
		Phone:        cfg.AdminPhone,
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
		ReferralCode: idgen.ReferralCode(),
	})
	return err
}
