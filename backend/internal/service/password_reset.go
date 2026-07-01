package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/email"
	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/store"
)

// resetTokenTTL — link đặt lại mật khẩu sống 1 giờ rồi hết hạn.
const resetTokenTTL = time.Hour

// PasswordResetService: quên/đặt lại mật khẩu qua email (Resend). Token gửi trong link,
// chỉ lưu HASH; dùng 1 lần; hết hạn sau resetTokenTTL. Cấu hình Resend do admin nhập ở
// Thiết lập — CHƯA cấu hình thì tính năng không hoạt động.
type PasswordResetService struct {
	store    *store.Store
	settings *SettingsService
}

func NewPasswordResetService(s *store.Store, settings *SettingsService) *PasswordResetService {
	return &PasswordResetService{store: s, settings: settings}
}

// errResetUnavailable — Resend chưa cấu hình.
func errResetUnavailable() error {
	return errors.Join(ErrValidation, errors.New("tính năng đặt lại mật khẩu chưa được cấu hình (cần admin nhập Resend API key & email gửi ở Thiết lập)"))
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// newToken sinh token ngẫu nhiên 32 byte -> 64 hex.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// genReadablePassword sinh mật khẩu ngẫu nhiên dễ đọc (bỏ ký tự dễ nhầm 0/O/1/l/I).
func genReadablePassword() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return "HK" + string(out), nil // prefix đảm bảo độ dài ≥8 & dễ nhận biết
}

// AdminSetNewPassword: admin đặt TRỰC TIẾP mật khẩu mới (ngẫu nhiên) cho user và TRẢ VỀ plaintext
// để admin gửi cho người dùng. Không cần email/Resend. Vô hiệu hoá các token reset cũ.
func (s *PasswordResetService) AdminSetNewPassword(ctx context.Context, admin, target uuid.UUID) (string, error) {
	user, err := s.store.GetUserByID(ctx, target)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	pw, err := genReadablePassword()
	if err != nil {
		return "", err
	}
	hash, err := security.HashPassword(pw)
	if err != nil {
		return "", err
	}
	if err := s.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: user.ID, PasswordHash: hash}); err != nil {
		return "", err
	}
	_ = s.store.DeleteUnusedPasswordResetTokens(ctx, user.ID)
	if err := audit.Write(ctx, s.store.Queries, audit.Actor(admin), "user.admin_set_password", "users", target.String(), nil, nil); err != nil {
		return "", err
	}
	return pw, nil
}

// mailer dựng Resend client từ cấu hình hiện hành; ok=false nếu chưa cấu hình.
func (s *PasswordResetService) mailer(ctx context.Context) (*email.Resend, bool) {
	apiKey, fromEmail, fromName, ok := s.settings.ResendConfig(ctx)
	if !ok {
		return nil, false
	}
	return email.NewResend(apiKey, fromEmail, fromName)
}

// resetLink dựng URL đặt lại từ baseURL (đã bỏ "/" cuối) + token.
func resetLink(baseURL, token string) string {
	return strings.TrimRight(baseURL, "/") + "/reset-password?token=" + token
}

// issueAndSend: vô hiệu token cũ, tạo token mới, gửi email. Dùng chung cho self-service & admin.
func (s *PasswordResetService) issueAndSend(ctx context.Context, m *email.Resend, user db.User, baseURL string) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	if err := s.store.DeleteUnusedPasswordResetTokens(ctx, user.ID); err != nil {
		return err
	}
	if err := s.store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: sha256Hex(token),
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}); err != nil {
		return err
	}
	link := resetLink(baseURL, token)
	subject := "Đặt lại mật khẩu — HKGROUP"
	body := fmt.Sprintf(`<div style="margin:0;padding:24px 0;background:#faf8f3">
<div style="max-width:520px;margin:0 auto;background:#ffffff;border:1px solid #eee7d8;border-radius:16px;overflow:hidden;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;color:#333">
  <div style="height:4px;background:linear-gradient(90deg,#b78a3c,#e0c079)"></div>
  <div style="padding:28px 32px 6px;text-align:center">
    <div style="font-size:22px;font-weight:700;color:#1f3d2a;letter-spacing:1px">HKGROUP</div>
    <div style="font-size:12px;color:#a99f86;letter-spacing:2px;text-transform:uppercase;margin-top:2px">Dược liệu lên men</div>
  </div>
  <div style="padding:14px 36px 28px">
    <h2 style="margin:0 0 14px;font-size:18px;color:#1f3d2a;font-weight:600;text-align:center">Đặt lại mật khẩu</h2>
    <p style="margin:0 0 6px;font-size:14px;color:#555">Xin chào %s,</p>
    <p style="margin:0 0 4px;font-size:14px;color:#555;line-height:1.7">Chúng tôi nhận được yêu cầu đặt lại mật khẩu cho tài khoản của bạn. Bấm nút bên dưới để tạo mật khẩu mới — liên kết có hiệu lực trong <b>1 giờ</b> và chỉ dùng một lần.</p>
    <div style="text-align:center;margin:26px 0">
      <a href="%s" style="background:#1f3d2a;color:#e8c877;text-decoration:none;padding:13px 30px;border-radius:9999px;font-weight:600;font-size:14px;display:inline-block">Đặt lại mật khẩu</a>
    </div>
    <p style="font-size:12px;color:#9a958a;text-align:center;word-break:break-all">Hoặc mở liên kết:<br/><a href="%s" style="color:#b78a3c">%s</a></p>
    <p style="margin:18px 0 0;font-size:12px;color:#a99f86;text-align:center;line-height:1.7">Nếu bạn không yêu cầu, hãy bỏ qua email này — mật khẩu của bạn không thay đổi.</p>
  </div>
  <div style="background:#1f3d2a;padding:14px 24px;font-size:12px;color:#c9d3c9;text-align:center">© HKGROUP · duoclieuhk.vn</div>
</div></div>`, html.EscapeString(user.FullName), link, link, link)

	return m.Send(ctx, user.Email, subject, body)
}

// RequestReset — self-service "Quên mật khẩu?": gửi link đặt lại tới email người dùng.
// CHỐNG DÒ EMAIL: nếu email không tồn tại vẫn trả nil (không tiết lộ). Nếu Resend chưa
// cấu hình thì trả lỗi rõ ràng (tính năng không hoạt động).
func (s *PasswordResetService) RequestReset(ctx context.Context, emailAddr, baseURL string) error {
	m, ok := s.mailer(ctx)
	if !ok {
		return errResetUnavailable()
	}
	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))
	if emailAddr == "" {
		return ErrValidation
	}
	user, err := s.store.GetUserByEmail(ctx, emailAddr)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // không tiết lộ email có tồn tại hay không
	}
	if err != nil {
		return err
	}
	return s.issueAndSend(ctx, m, user, baseURL)
}

// AdminRequestReset — admin chủ động gửi link đặt lại cho 1 tài khoản. Khác self-service:
// báo lỗi nếu user không tồn tại, và ghi audit với actor = admin.
func (s *PasswordResetService) AdminRequestReset(ctx context.Context, admin, target uuid.UUID, baseURL string) error {
	m, ok := s.mailer(ctx)
	if !ok {
		return errResetUnavailable()
	}
	user, err := s.store.GetUserByID(ctx, target)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.issueAndSend(ctx, m, user, baseURL); err != nil {
		return err
	}
	return audit.Write(ctx, s.store.Queries, audit.Actor(admin), "user.admin_reset_password", "users", target.String(), nil, nil)
}

// ResetPassword — đổi mật khẩu bằng token từ link. Kiểm tra token tồn tại, chưa dùng,
// chưa hết hạn; đặt mật khẩu mới; đánh dấu token đã dùng + vô hiệu các token còn lại.
func (s *PasswordResetService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.Join(ErrValidation, errors.New("mật khẩu mới phải có ít nhất 8 ký tự"))
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.Join(ErrValidation, errors.New("liên kết không hợp lệ"))
	}
	row, err := s.store.GetPasswordResetTokenByHash(ctx, sha256Hex(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.Join(ErrValidation, errors.New("liên kết không hợp lệ"))
	}
	if err != nil {
		return err
	}
	if row.UsedAt.Valid {
		return errors.Join(ErrValidation, errors.New("liên kết đã được sử dụng"))
	}
	if !row.ExpiresAt.Valid || time.Now().After(row.ExpiresAt.Time) {
		return errors.Join(ErrValidation, errors.New("liên kết đã hết hạn — vui lòng yêu cầu lại"))
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: row.UserID, PasswordHash: hash}); e != nil {
			return e
		}
		if e := q.MarkPasswordResetTokenUsed(ctx, row.ID); e != nil {
			return e
		}
		if e := q.DeleteUnusedPasswordResetTokens(ctx, row.UserID); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(row.UserID), "user.reset_password", "users", row.UserID.String(), nil, nil)
	})
}
