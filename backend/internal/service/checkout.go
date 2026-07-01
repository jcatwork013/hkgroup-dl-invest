package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/email"
	"github.com/hkgroup/backend/internal/platform/idgen"
	"github.com/hkgroup/backend/internal/platform/security"
)

// PublicCheckoutInput — đơn mua hàng online của KHÁCH (không đăng nhập) trên duoclieuhk.vn.
type PublicCheckoutInput struct {
	CustomerName  string           `json:"customer_name"`
	CustomerPhone string           `json:"customer_phone"`
	Address       string           `json:"address"`
	Email         string           `json:"email"` // gửi hoá đơn nếu có
	Note          string           `json:"note"`
	RefCode       string           `json:"ref_code"` // mã giới thiệu affiliate (tuỳ chọn, từ ?ref=)
	Items         []OrderItemInput `json:"items"`
}

type CheckoutBank struct {
	BankName    string `json:"bank_name"`
	BankCode    string `json:"bank_code"`
	Account     string `json:"account"`
	AccountName string `json:"account_name"`
}

type CheckoutResult struct {
	Code        string       `json:"code"`
	SubtotalVnd int64        `json:"subtotal_vnd"`
	Status      string       `json:"status"`
	Bank        CheckoutBank `json:"bank"`
}

const houseSellerEmail = "online@duoclieuhk.vn"

// ensureHouseSeller trả về tài khoản saler "Bán hàng Online" dùng làm người bán cho đơn khách tự đặt.
// Ưu tiên setting sales_house_seller_id; nếu chưa có thì tìm/ tạo tài khoản online@ rồi lưu lại.
func (s *SalesService) ensureHouseSeller(ctx context.Context) (uuid.UUID, error) {
	if idStr := s.settings.Str(ctx, "sales_house_seller_id", ""); idStr != "" {
		if uid, err := uuid.Parse(idStr); err == nil {
			if u, e := s.store.GetUserByID(ctx, uid); e == nil && u.Role == db.UserRoleSaler {
				return uid, nil
			}
		}
	}
	if u, e := s.store.GetUserByEmail(ctx, houseSellerEmail); e == nil {
		s.saveHouseSeller(ctx, u.ID)
		return u.ID, nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return uuid.Nil, e
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return uuid.Nil, err
	}
	hash, err := security.HashPassword("HOUSE-" + hex.EncodeToString(buf))
	if err != nil {
		return uuid.Nil, err
	}
	u, err := s.store.CreateUser(ctx, db.CreateUserParams{
		FullName:     "Bán hàng Online",
		Phone:        "0000000000",
		Email:        houseSellerEmail,
		PasswordHash: hash,
		Role:         db.UserRoleSaler,
		ReferralCode: idgen.ReferralCode(),
	})
	if err != nil {
		return uuid.Nil, err
	}
	s.saveHouseSeller(ctx, u.ID)
	return u.ID, nil
}

func (s *SalesService) saveHouseSeller(ctx context.Context, id uuid.UUID) {
	_, _ = s.store.UpsertSetting(ctx, db.UpsertSettingParams{Key: "sales_house_seller_id", Value: id.String()})
}

// PublicCheckout tạo đơn hàng PENDING từ giỏ hàng của khách. Giá tính từ SẢN PHẨM (server-trusted),
// không tin client. Trả mã đơn + thông tin chuyển khoản để khách thanh toán; admin xác nhận sau.
func (s *SalesService) PublicCheckout(ctx context.Context, in PublicCheckoutInput) (CheckoutResult, error) {
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	if in.CustomerName == "" || in.CustomerPhone == "" {
		return CheckoutResult{}, errors.Join(ErrValidation, errors.New("cần họ tên và số điện thoại"))
	}
	if len(in.Items) == 0 {
		return CheckoutResult{}, errors.Join(ErrValidation, errors.New("giỏ hàng trống"))
	}
	seller, err := s.ensureHouseSeller(ctx)
	if err != nil {
		return CheckoutResult{}, err
	}
	// Gán affiliate theo cơ chế KHOÁ FIRST-TOUCH (theo SĐT khách):
	//  1) Nếu khách ĐÃ bị khoá với 1 affiliate → luôn dùng affiliate đó (bỏ qua ref_code mới).
	//  2) Nếu CHƯA khoá và ref_code hợp lệ → gán affiliate đó + KHOÁ vĩnh viễn cho SĐT này.
	affiliateID := ""
	if lockedID, e := s.store.GetReferralLock(ctx, in.CustomerPhone); e == nil {
		if lockedID != seller {
			affiliateID = lockedID.String()
		}
	} else if errors.Is(e, pgx.ErrNoRows) {
		if code := strings.TrimSpace(in.RefCode); code != "" {
			if u, e := s.store.GetUserByReferralCode(ctx, code); e == nil && u.ID != seller && u.Role == db.UserRoleSaler {
				affiliateID = u.ID.String()
				// Khoá first-touch: từ nay mọi đơn của SĐT này về đúng affiliate này.
				_ = s.store.SetReferralLockIfAbsent(ctx, in.CustomerPhone, u.ID)
			}
		}
	}
	note := strings.TrimSpace(in.Note)
	if addr := strings.TrimSpace(in.Address); addr != "" {
		note = "Địa chỉ: " + addr
		if in.Note != "" {
			note += "\nGhi chú: " + strings.TrimSpace(in.Note)
		}
	}
	order, err := s.CreateOrder(ctx, seller, false, OrderInput{
		CustomerName:  in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		AffiliateID:   affiliateID,
		Note:          note,
		Items:         in.Items,
	})
	if err != nil {
		return CheckoutResult{}, err
	}
	bankName, account, accountName, _ := s.settings.CompanyBank(ctx)
	bankCode := s.settings.Str(ctx, "company_bank_code", "")

	// Gửi hoá đơn qua email nếu khách cung cấp (best-effort, không chặn đặt hàng).
	if em := strings.TrimSpace(in.Email); em != "" {
		s.sendInvoiceEmail(ctx, em, order, in, bankName, account, accountName, bankCode)
	}

	return CheckoutResult{
		Code:        order.Code,
		SubtotalVnd: order.SubtotalVnd,
		Status:      string(order.Status),
		Bank: CheckoutBank{
			BankName:    bankName,
			BankCode:    bankCode,
			Account:     account,
			AccountName: accountName,
		},
	}, nil
}

// sendInvoiceEmail gửi hoá đơn HTML qua Resend (nếu admin đã cấu hình). Không trả lỗi ra ngoài.
func (s *SalesService) sendInvoiceEmail(ctx context.Context, to string, order db.SalesOrder, in PublicCheckoutInput, bankName, account, accountName, bankCode string) {
	apiKey, fromEmail, fromName, ok := s.settings.ResendConfig(ctx)
	if !ok {
		return
	}
	mailer, ok := email.NewResend(apiKey, fromEmail, fromName)
	if !ok {
		return
	}
	var rows strings.Builder
	for _, it := range in.Items {
		pid, e := uuid.Parse(it.ProductID)
		if e != nil {
			continue
		}
		p, e := s.store.GetProduct(ctx, pid)
		if e != nil {
			continue
		}
		rows.WriteString(fmt.Sprintf(
			`<tr><td style="padding:6px 4px;border-bottom:1px solid #eee">%s</td><td align="center" style="padding:6px 4px;border-bottom:1px solid #eee">%d</td><td align="right" style="padding:6px 4px;border-bottom:1px solid #eee">%s đ</td></tr>`,
			html.EscapeString(p.Name), it.Qty, groupVND(p.PriceVnd*it.Qty),
		))
	}
	hotline := s.settings.Str(ctx, "contact_hotline", "")
	supEmail := s.settings.Str(ctx, "contact_email", "")
	qr := ""
	if bankCode != "" && account != "" {
		qr = fmt.Sprintf(
			`<div style="text-align:center;padding:6px 0 14px"><img src="https://img.vietqr.io/image/%s-%s-compact2.png?amount=%d&addInfo=%s&accountName=%s" alt="QR" width="176" height="176" style="border-radius:10px;border:1px solid #eae4d6"/><div style="font-size:12px;color:#9a958a;margin-top:6px">Quét mã để chuyển khoản nhanh</div></div>`,
			bankCode, account, order.SubtotalVnd, "Mua%20san%20pham%20"+html.EscapeString(order.Code), html.EscapeString(accountName))
	}
	contact := ""
	if hotline != "" || supEmail != "" {
		parts := ""
		if hotline != "" {
			parts += `Hotline <b style="color:#1f3d2a">` + html.EscapeString(hotline) + `</b>`
		}
		if supEmail != "" {
			if parts != "" {
				parts += ` &nbsp;·&nbsp; `
			}
			parts += `Email <b style="color:#1f3d2a">` + html.EscapeString(supEmail) + `</b>`
		}
		contact = `<p style="margin:0;font-size:13px;color:#6b6b6b;text-align:center">Cần hỗ trợ? Liên hệ ` + parts + `</p>`
	}

	body := fmt.Sprintf(`<div style="margin:0;padding:24px 0;background:#faf8f3">
<div style="max-width:560px;margin:0 auto;background:#ffffff;border:1px solid #eee7d8;border-radius:16px;overflow:hidden;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;color:#333">
  <div style="height:4px;background:linear-gradient(90deg,#b78a3c,#e0c079)"></div>
  <div style="padding:28px 32px 8px;text-align:center">
    <div style="font-size:22px;font-weight:700;color:#1f3d2a;letter-spacing:1px">HKGROUP</div>
    <div style="font-size:12px;color:#a99f86;letter-spacing:2px;text-transform:uppercase;margin-top:2px">Dược liệu lên men</div>
  </div>
  <div style="padding:12px 32px 4px;text-align:center">
    <h2 style="margin:0;font-size:19px;color:#1f3d2a;font-weight:600">Cảm ơn bạn đã đặt hàng</h2>
    <p style="margin:8px 0 0;font-size:14px;color:#777">Xin chào %s, đơn hàng <b style="color:#1f3d2a">%s</b> của bạn đã được ghi nhận.</p>
  </div>

  <div style="padding:20px 32px 4px">
    <table style="width:100%%;border-collapse:separate;border-spacing:0;font-size:13px;color:#888;text-align:center"><tr>
      <td style="padding:6px 2px"><div style="color:#1f3d2a;font-weight:700">1 ✓</div>Đặt hàng</td>
      <td style="color:#dcd6c8">→</td>
      <td style="padding:6px 2px"><div style="color:#b78a3c;font-weight:700">2</div>Chuyển khoản</td>
      <td style="color:#dcd6c8">→</td>
      <td style="padding:6px 2px"><div style="color:#bbb;font-weight:700">3</div>Xác nhận</td>
      <td style="color:#dcd6c8">→</td>
      <td style="padding:6px 2px"><div style="color:#bbb;font-weight:700">4</div>Giao hàng</td>
    </tr></table>
  </div>

  <div style="padding:16px 32px 0">
    <table style="width:100%%;border-collapse:collapse;font-size:14px">
      <thead><tr>
        <th align="left" style="padding:8px 0;border-bottom:1px solid #eee;color:#a99f86;font-weight:600;font-size:12px;text-transform:uppercase;letter-spacing:.5px">Sản phẩm</th>
        <th align="center" style="padding:8px 0;border-bottom:1px solid #eee;color:#a99f86;font-weight:600;font-size:12px">SL</th>
        <th align="right" style="padding:8px 0;border-bottom:1px solid #eee;color:#a99f86;font-weight:600;font-size:12px">Thành tiền</th>
      </tr></thead>
      <tbody>%s</tbody>
    </table>
    <div style="text-align:right;font-size:16px;font-weight:700;margin-top:14px;color:#1f3d2a">Tổng cộng: %s đ</div>
  </div>

  <div style="margin:22px 32px;background:#faf8f3;border:1px solid #f0ebdd;border-radius:12px;padding:18px 20px">
    <div style="font-size:12px;text-transform:uppercase;letter-spacing:1px;color:#b78a3c;font-weight:700;text-align:center;margin-bottom:8px">Thanh toán chuyển khoản</div>
    %s
    <div style="font-size:14px;color:#555;line-height:1.9">
      Ngân hàng: <b style="color:#333">%s</b><br/>Số tài khoản: <b style="color:#333">%s</b><br/>Chủ tài khoản: <b style="color:#333">%s</b><br/>Nội dung CK: <b style="color:#1f3d2a">Mua san pham %s</b>
    </div>
  </div>

  <div style="padding:0 32px 8px">
    <p style="margin:0 0 10px;font-size:13px;color:#777;text-align:center;line-height:1.7">Sau khi nhận được thanh toán, chúng tôi sẽ liên hệ xác nhận và tiến hành giao hàng trong 1–5 ngày làm việc.</p>
    %s
  </div>

  <div style="background:#1f3d2a;padding:16px 24px;font-size:12px;color:#c9d3c9;text-align:center;margin-top:16px">© HKGROUP · duoclieuhk.vn — Dược liệu lên men từ thiên nhiên Việt</div>
</div></div>`,
		html.EscapeString(in.CustomerName), html.EscapeString(order.Code), rows.String(),
		groupVND(order.SubtotalVnd), qr, html.EscapeString(bankName), html.EscapeString(account),
		html.EscapeString(accountName), html.EscapeString(order.Code), contact)
	_ = mailer.Send(ctx, to, "Hoá đơn "+order.Code+" — HKGROUP", body)
}

// groupVND format số tiền VND có dấu chấm ngăn cách nghìn (server-side, cho email).
func groupVND(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteRune(c)
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}
