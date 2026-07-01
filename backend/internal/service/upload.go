package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/store"
)

// UploadService stores confidential KYC images AES-256-GCM-encrypted at rest.
type UploadService struct {
	store   *store.Store
	crypto  *security.Cryptor
	baseDir string
}

func NewUploadService(s *store.Store, crypto *security.Cryptor, baseDir string) (*UploadService, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, err
	}
	return &UploadService{store: s, crypto: crypto, baseDir: baseDir}, nil
}

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true, // vài trình duyệt gửi jpg thay vì jpeg
	"image/png":  true,
	"image/webp": true,
	"image/heic": true,
	"image/heif": true, // ảnh iPhone (HEIC) đôi khi content-type là heif
	"image/gif":  true,
}

// Save encrypts and stores an image, returns the upload row.
func (s *UploadService) Save(ctx context.Context, userID uuid.UUID, kind, contentType string, data []byte) (db.Upload, error) {
	if !allowedImageTypes[contentType] {
		return db.Upload{}, errors.Join(ErrValidation, errors.New("chỉ chấp nhận ảnh JPEG/PNG/WEBP/HEIC"))
	}
	if len(data) == 0 || len(data) > 12*1024*1024 {
		return db.Upload{}, errors.Join(ErrValidation, errors.New("ảnh rỗng hoặc quá lớn (tối đa 12MB)"))
	}
	enc, err := s.crypto.Encrypt(data)
	if err != nil {
		return db.Upload{}, err
	}
	id := uuid.New()
	path := filepath.Join(s.baseDir, id.String()+".enc")
	if err := os.WriteFile(path, enc, 0o600); err != nil {
		return db.Upload{}, err
	}
	up, err := s.store.CreateUpload(ctx, db.CreateUploadParams{
		UserID: userID, Kind: kind, ContentType: contentType, Path: path,
	})
	if err != nil {
		_ = os.Remove(path)
		return db.Upload{}, err
	}
	// Force the generated id to match the row (CreateUpload generated its own; use the row's id/path).
	return up, nil
}

// SaveProductImage stores a PUBLIC product image UNENCRYPTED (customers must see it). Kind="product".
func (s *UploadService) SaveProductImage(ctx context.Context, admin uuid.UUID, contentType string, data []byte) (db.Upload, error) {
	if !allowedImageTypes[contentType] {
		return db.Upload{}, errors.Join(ErrValidation, errors.New("chỉ chấp nhận ảnh JPEG/PNG/WEBP/HEIC"))
	}
	if len(data) == 0 || len(data) > 12*1024*1024 {
		return db.Upload{}, errors.Join(ErrValidation, errors.New("ảnh rỗng hoặc quá lớn (tối đa 12MB)"))
	}
	dir := filepath.Join(s.baseDir, "products")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return db.Upload{}, err
	}
	id := uuid.New()
	path := filepath.Join(dir, id.String())
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return db.Upload{}, err
	}
	up, err := s.store.CreateUpload(ctx, db.CreateUploadParams{
		UserID: admin, Kind: "product", ContentType: contentType, Path: path,
	})
	if err != nil {
		_ = os.Remove(path)
		return db.Upload{}, err
	}
	return up, nil
}

// LoadPublicImage returns a PUBLIC product image (no auth, no decryption). Only kind="product".
func (s *UploadService) LoadPublicImage(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	up, err := s.store.GetUpload(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if up.Kind != "product" {
		return nil, "", ErrForbidden // chỉ phục vụ ảnh sản phẩm công khai; KYC vẫn private
	}
	data, err := os.ReadFile(up.Path)
	if err != nil {
		return nil, "", ErrNotFound
	}
	return data, up.ContentType, nil
}

// Load returns the decrypted bytes + content type. Access: owner or admin only.
func (s *UploadService) Load(ctx context.Context, id, requester uuid.UUID, isAdmin bool) ([]byte, string, error) {
	up, err := s.store.GetUpload(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if !isAdmin && up.UserID != requester {
		return nil, "", ErrForbidden
	}
	enc, err := os.ReadFile(up.Path)
	if err != nil {
		return nil, "", ErrNotFound
	}
	plain, err := s.crypto.Decrypt(enc)
	if err != nil {
		return nil, "", err
	}
	return plain, up.ContentType, nil
}
