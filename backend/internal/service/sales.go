package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

// SalesService quản lý CATALOG bán hàng (danh mục + sản phẩm). Đơn hàng & chia hoa hồng
// nằm ở phần mở rộng (phase 2) nhưng cùng service này để gom nghiệp vụ bán hàng.
type SalesService struct {
	store    *store.Store
	settings *SettingsService
}

func NewSalesService(s *store.Store, settings *SettingsService) *SalesService {
	return &SalesService{store: s, settings: settings}
}

var slugNonWord = regexp.MustCompile(`[^a-z0-9]+`)

// slugify tạo slug ASCII đơn giản từ tên (đủ dùng làm khoá danh mục).
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonWord.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// ----------------------------- Danh mục -----------------------------

type CategoryInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
	Active      bool   `json:"active"`
}

func (s *SalesService) ListCategories(ctx context.Context) ([]db.ProductCategory, error) {
	return s.store.ListProductCategories(ctx)
}

func (s *SalesService) CreateCategory(ctx context.Context, admin uuid.UUID, in CategoryInput) (db.ProductCategory, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return db.ProductCategory{}, ErrValidation
	}
	if in.Slug == "" {
		in.Slug = slugify(in.Name)
	}
	var c db.ProductCategory
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		c, e = q.CreateProductCategory(ctx, db.CreateProductCategoryParams{
			Name: in.Name, Slug: in.Slug, Description: in.Description, SortOrder: in.SortOrder, Active: in.Active,
		})
		if isUniqueViolation(e) {
			return ErrConflict
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "category.create", "product_categories", c.ID.String(), nil, c)
	})
	return c, err
}

func (s *SalesService) UpdateCategory(ctx context.Context, admin, id uuid.UUID, in CategoryInput) (db.ProductCategory, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return db.ProductCategory{}, ErrValidation
	}
	if in.Slug == "" {
		in.Slug = slugify(in.Name)
	}
	var c db.ProductCategory
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		c, e = q.UpdateProductCategory(ctx, db.UpdateProductCategoryParams{
			ID: id, Name: in.Name, Slug: in.Slug, Description: in.Description, SortOrder: in.SortOrder, Active: in.Active,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if isUniqueViolation(e) {
			return ErrConflict
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "category.update", "product_categories", id.String(), nil, c)
	})
	return c, err
}

func (s *SalesService) DeleteCategory(ctx context.Context, admin, id uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.DeleteProductCategory(ctx, id); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "category.delete", "product_categories", id.String(), nil, nil)
	})
}

// ----------------------------- Sản phẩm -----------------------------

type ProductInput struct {
	CategoryID   string `json:"category_id"`
	Sku          string `json:"sku"`
	Name         string `json:"name"`
	Badge        string `json:"badge"`
	PriceVnd     int64  `json:"price_vnd"`
	CostVnd      int64  `json:"cost_vnd"`
	ImageUrl     string `json:"image_url"`
	Summary      string `json:"summary"`
	Description  string `json:"description"`
	SpecWarranty string `json:"spec_warranty"`
	SpecTrace    string `json:"spec_trace"`
	SpecDelivery string `json:"spec_delivery"`
	SpecReturn   string `json:"spec_return"`
	Active       bool   `json:"active"`
}

func (in ProductInput) categoryUUID() uuid.NullUUID {
	if id, err := uuid.Parse(in.CategoryID); err == nil {
		return uuid.NullUUID{UUID: id, Valid: true}
	}
	return uuid.NullUUID{}
}

func (in ProductInput) valid() bool {
	return strings.TrimSpace(in.Name) != "" && in.PriceVnd >= 0 && in.CostVnd >= 0
}

func (s *SalesService) ListProducts(ctx context.Context) ([]db.Product, error) {
	return s.store.ListProducts(ctx)
}

func (s *SalesService) GetProduct(ctx context.Context, id uuid.UUID) (db.Product, error) {
	p, err := s.store.GetProduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Product{}, ErrNotFound
	}
	return p, err
}

// withSpecDefaults áp giá trị mặc định 4 thông số nếu admin để trống.
func (in *ProductInput) withSpecDefaults() {
	if in.SpecWarranty == "" {
		in.SpecWarranty = "Chính hãng 100%"
	}
	if in.SpecTrace == "" {
		in.SpecTrace = "Theo từng lô"
	}
	if in.SpecDelivery == "" {
		in.SpecDelivery = "Hub theo khu vực"
	}
	if in.SpecReturn == "" {
		in.SpecReturn = "Trong 7 ngày"
	}
}

func (s *SalesService) CreateProduct(ctx context.Context, admin uuid.UUID, in ProductInput) (db.Product, error) {
	if !in.valid() {
		return db.Product{}, ErrValidation
	}
	in.withSpecDefaults()
	if strings.TrimSpace(in.Sku) == "" {
		in.Sku = "SP-" + slugify(in.Name)
	}
	var p db.Product
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		p, e = q.CreateProduct(ctx, db.CreateProductParams{
			CategoryID: in.categoryUUID(), Sku: in.Sku, Name: strings.TrimSpace(in.Name), Badge: in.Badge,
			PriceVnd: in.PriceVnd, CostVnd: in.CostVnd, ImageUrl: in.ImageUrl, Summary: in.Summary,
			Description: in.Description, SpecWarranty: in.SpecWarranty, SpecTrace: in.SpecTrace,
			SpecDelivery: in.SpecDelivery, SpecReturn: in.SpecReturn, Active: in.Active,
		})
		if isUniqueViolation(e) {
			return ErrConflict
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "product.create", "products", p.ID.String(), nil, p)
	})
	return p, err
}

func (s *SalesService) UpdateProduct(ctx context.Context, admin, id uuid.UUID, in ProductInput) (db.Product, error) {
	if !in.valid() {
		return db.Product{}, ErrValidation
	}
	in.withSpecDefaults()
	if strings.TrimSpace(in.Sku) == "" {
		in.Sku = "SP-" + slugify(in.Name)
	}
	var p db.Product
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		p, e = q.UpdateProduct(ctx, db.UpdateProductParams{
			ID: id, CategoryID: in.categoryUUID(), Sku: in.Sku, Name: strings.TrimSpace(in.Name), Badge: in.Badge,
			PriceVnd: in.PriceVnd, CostVnd: in.CostVnd, ImageUrl: in.ImageUrl, Summary: in.Summary,
			Description: in.Description, SpecWarranty: in.SpecWarranty, SpecTrace: in.SpecTrace,
			SpecDelivery: in.SpecDelivery, SpecReturn: in.SpecReturn, Active: in.Active,
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if isUniqueViolation(e) {
			return ErrConflict
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "product.update", "products", id.String(), nil, p)
	})
	return p, err
}

func (s *SalesService) DeleteProduct(ctx context.Context, admin, id uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.DeleteProduct(ctx, id); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "product.delete", "products", id.String(), nil, nil)
	})
}
