// Package email gửi email giao dịch qua Resend (https://resend.com) bằng HTTP API.
// Cấu hình (API key, địa chỉ + tên người gửi) do admin nhập ở Thiết lập, lưu trong
// site_settings — KHÔNG hard-code. Nếu chưa cấu hình thì tính năng phụ thuộc (đặt lại
// mật khẩu) sẽ không hoạt động.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const resendEndpoint = "https://api.resend.com/emails"

// Resend là client tối giản gọi Resend API.
type Resend struct {
	APIKey    string
	FromEmail string
	FromName  string
	client    *http.Client
}

// NewResend tạo client từ cấu hình. ok=false nếu thiếu API key hoặc địa chỉ gửi —
// gọi nơi dùng phải kiểm tra ok trước khi Send.
func NewResend(apiKey, fromEmail, fromName string) (*Resend, bool) {
	apiKey = strings.TrimSpace(apiKey)
	fromEmail = strings.TrimSpace(fromEmail)
	if apiKey == "" || fromEmail == "" {
		return nil, false
	}
	return &Resend{
		APIKey:    apiKey,
		FromEmail: fromEmail,
		FromName:  strings.TrimSpace(fromName),
		client:    &http.Client{Timeout: 15 * time.Second},
	}, true
}

// from dựng header From dạng "Tên <email>" nếu có tên, ngược lại chỉ email.
func (r *Resend) from() string {
	if r.FromName != "" {
		return fmt.Sprintf("%s <%s>", r.FromName, r.FromEmail)
	}
	return r.FromEmail
}

type sendBody struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// Send gửi 1 email HTML tới 1 người nhận. Trả lỗi nếu Resend từ chối (vd API key sai,
// domain chưa verify) để nơi gọi báo lại cho admin.
func (r *Resend) Send(ctx context.Context, to, subject, html string) error {
	payload, err := json.Marshal(sendBody{From: r.from(), To: []string{to}, Subject: subject, HTML: html})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEndpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("resend trả về %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
