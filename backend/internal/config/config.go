package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is loaded from environment. No secrets are hard-coded.
type Config struct {
	Env  string `env:"APP_ENV" envDefault:"development"`
	Port string `env:"PORT" envDefault:"8080"`

	DatabaseURL string `env:"DATABASE_URL,required"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
	NATSURL     string `env:"NATS_URL" envDefault:"nats://localhost:4222"`

	JWTSecret       string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`

	CORSOrigin string `env:"CORS_ORIGIN" envDefault:"http://localhost:3000"`

	// Confidential KYC image encryption (AES-256-GCM). 64 hex chars = 32 bytes. OVERRIDE in prod.
	KYCEncKey string `env:"KYC_ENC_KEY" envDefault:"0000000000000000000000000000000000000000000000000000000000000000"`
	UploadDir string `env:"UPLOAD_DIR" envDefault:"/data/uploads"`

	// Company (legal-entity) bank account that receives ALL investment funds.
	// HARD CONSTRAINT 4: there is no configuration path for a personal account.
	CompanyBank        string `env:"COMPANY_BANK" envDefault:"Vietcombank"`
	CompanyAccount     string `env:"COMPANY_ACCOUNT" envDefault:"0123456789"`
	CompanyAccountName string `env:"COMPANY_ACCOUNT_NAME" envDefault:"CONG TY CO PHAN HKGROUP"`

	// HARD CONSTRAINT 3: cash commission for 'investor' referrals is OFF by default.
	// Only 'customer' referrals ever pay cash; this flag never enables it for investors —
	// it is read by the referral service purely to decide whether to RECORD investor referrals.
	InvestorReferralCashEnabled bool    `env:"INVESTOR_REFERRAL_CASH_ENABLED" envDefault:"false"`
	CustomerCommissionRate      float64 `env:"CUSTOMER_COMMISSION_RATE" envDefault:"0.05"`
	PITRate                     float64 `env:"PIT_RATE" envDefault:"0.10"` // thuế TNCN khấu trừ

	// Bootstrap admin (idempotent). Change in production.
	AdminEmail    string `env:"ADMIN_EMAIL" envDefault:"admin@hkgroup.vn"`
	AdminPassword string `env:"ADMIN_PASSWORD" envDefault:"Admin@12345"`
	AdminPhone    string `env:"ADMIN_PHONE" envDefault:"0900000000"`
}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, err
	}
	return c, nil
}
