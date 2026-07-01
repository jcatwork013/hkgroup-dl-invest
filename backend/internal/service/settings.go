package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

// SettingsService manages editable site-wide settings (contact info, brand year...).
type SettingsService struct {
	store *store.Store
}

func NewSettingsService(s *store.Store) *SettingsService { return &SettingsService{store: s} }

// allowedKeys whitelists which settings admins may edit / are exposed publicly.
var allowedKeys = map[string]bool{
	"contact_hotline": true,
	"contact_address": true,
	"contact_email":   true,
	"brand_since":     true,
	// Company (legal-entity) receiving account — HARD CONSTRAINT 4: company account only.
	"company_bank_code":    true,
	"company_bank_name":    true,
	"company_account":      true,
	"company_account_name": true,
	// Revenue distribution + referral F1/F2/F3 + public pool toggle.
	"pool_rate":              true,
	"investor_share_rate":    true,
	"referral_f1_rate":       true,
	"referral_f2_rate":       true,
	"referral_f3_rate":       true,
	"referral_investor_cash": true,
	"show_pool_public":       true,
	// Tiered "đồng chia + bonus" distribution (9% equal + 6% band-bonus; all configurable).
	"dist_equal_rate":    true,
	"dist_bonus_rate":    true,
	"dist_band1_max":     true,
	"dist_band1_rate":    true,
	"dist_band2_max":     true,
	"dist_band2_rate":    true,
	"dist_band3_rate":    true,
	"dist_residual_mode": true, // "rollover" | "retain"
	// Lịch rút tiền — CSV các ngày trong tháng được phép gửi yêu cầu rút (vd "15,30").
	"withdrawal_days": true,
	// Gửi email (Resend) — phục vụ ĐẶT LẠI MẬT KHẨU. resend_api_key là BÍ MẬT (xem secretKeys).
	"resend_api_key":    true,
	"resend_from_email": true,
	"resend_from_name":  true,
	"app_base_url":      true, // URL gốc của web để dựng link đặt lại mật khẩu (vd https://duoclieuhk.vn)
}

// secretKeys: những setting KHÔNG bao giờ được trả ra ngoài (kể cả cho admin) — chỉ ghi.
// Endpoint công khai loại bỏ hẳn; endpoint admin chỉ báo "đã cấu hình hay chưa".
var secretKeys = map[string]bool{
	"resend_api_key": true,
}

// Str reads a string setting with a fallback default.
func (s *SettingsService) Str(ctx context.Context, key, def string) string {
	m, err := s.List(ctx)
	if err != nil {
		return def
	}
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

// Float reads a numeric setting with a fallback default.
func (s *SettingsService) Float(ctx context.Context, key string, def float64) float64 {
	m, err := s.List(ctx)
	if err != nil {
		return def
	}
	if v, ok := m[key]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// CompanyBank resolves the configured company receiving account from settings.
func (s *SettingsService) CompanyBank(ctx context.Context) (bankName, account, accountName string, ok bool) {
	m, err := s.List(ctx)
	if err != nil {
		return "", "", "", false
	}
	bankName = m["company_bank_name"]
	account = m["company_account"]
	accountName = m["company_account_name"]
	ok = account != "" && accountName != ""
	return bankName, account, accountName, ok
}

func (s *SettingsService) List(ctx context.Context) (map[string]string, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

// Public trả settings AN TOÀN để lộ công khai — loại bỏ mọi secret (vd resend_api_key).
func (s *SettingsService) Public(ctx context.Context) (map[string]string, error) {
	m, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for k := range secretKeys {
		delete(m, k)
	}
	return m, nil
}

// Admin trả settings cho trang quản trị: secret bị xoá GIÁ TRỊ (chỉ ghi, không đọc lại),
// kèm cờ "<key>_configured"=true/false để UI hiển thị trạng thái đã cấu hình hay chưa.
func (s *SettingsService) Admin(ctx context.Context) (map[string]string, error) {
	m, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for k := range secretKeys {
		configured := "false"
		if v, ok := m[k]; ok && v != "" {
			configured = "true"
		}
		delete(m, k)
		m[k+"_configured"] = configured
	}
	return m, nil
}

// ResendConfig đọc cấu hình gửi email. ok=false nếu thiếu API key hoặc địa chỉ gửi
// (khi đó tính năng đặt lại mật khẩu không hoạt động).
func (s *SettingsService) ResendConfig(ctx context.Context) (apiKey, fromEmail, fromName string, ok bool) {
	m, err := s.List(ctx)
	if err != nil {
		return "", "", "", false
	}
	apiKey = m["resend_api_key"]
	fromEmail = m["resend_from_email"]
	fromName = m["resend_from_name"]
	ok = apiKey != "" && fromEmail != ""
	return apiKey, fromEmail, fromName, ok
}

// Update upserts the given key/value pairs (admin only). Unknown keys are ignored.
func (s *SettingsService) Update(ctx context.Context, admin uuid.UUID, values map[string]string) (map[string]string, error) {
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		for k, v := range values {
			if !allowedKeys[k] {
				continue
			}
			// Secret (vd API key) là write-only: gửi rỗng = "giữ nguyên", KHÔNG ghi đè
			// để tránh việc lưu form bình thường vô tình xoá mất key đã cấu hình.
			if secretKeys[k] && strings.TrimSpace(v) == "" {
				continue
			}
			row, e := q.UpsertSetting(ctx, db.UpsertSettingParams{
				Key: k, Value: v, UpdatedBy: uuid.NullUUID{UUID: admin, Valid: true},
			})
			if e != nil {
				return e
			}
			if e = audit.Write(ctx, q, audit.Actor(admin), "settings.update", "site_settings", k, nil, row); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.List(ctx)
}
