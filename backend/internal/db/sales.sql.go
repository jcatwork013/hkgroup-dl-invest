// Hand-written to match sqlc output (sqlc generate is blocked by legacy reservation.sql).
// Queries for the sales module — phase 1: product categories + products.
package db

import (
	"context"

	"github.com/google/uuid"
)

// ============================ PRODUCT CATEGORIES ============================

const createProductCategory = `-- name: CreateProductCategory :one
INSERT INTO product_categories (name, slug, description, sort_order, active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, slug, description, sort_order, active, created_at, updated_at
`

type CreateProductCategoryParams struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
	Active      bool   `json:"active"`
}

func (q *Queries) CreateProductCategory(ctx context.Context, arg CreateProductCategoryParams) (ProductCategory, error) {
	row := q.db.QueryRow(ctx, createProductCategory, arg.Name, arg.Slug, arg.Description, arg.SortOrder, arg.Active)
	var i ProductCategory
	err := row.Scan(&i.ID, &i.Name, &i.Slug, &i.Description, &i.SortOrder, &i.Active, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const listProductCategories = `-- name: ListProductCategories :many
SELECT id, name, slug, description, sort_order, active, created_at, updated_at
FROM product_categories ORDER BY sort_order, name
`

func (q *Queries) ListProductCategories(ctx context.Context) ([]ProductCategory, error) {
	rows, err := q.db.Query(ctx, listProductCategories)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ProductCategory{}
	for rows.Next() {
		var i ProductCategory
		if err := rows.Scan(&i.ID, &i.Name, &i.Slug, &i.Description, &i.SortOrder, &i.Active, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const getProductCategory = `-- name: GetProductCategory :one
SELECT id, name, slug, description, sort_order, active, created_at, updated_at
FROM product_categories WHERE id = $1
`

func (q *Queries) GetProductCategory(ctx context.Context, id uuid.UUID) (ProductCategory, error) {
	row := q.db.QueryRow(ctx, getProductCategory, id)
	var i ProductCategory
	err := row.Scan(&i.ID, &i.Name, &i.Slug, &i.Description, &i.SortOrder, &i.Active, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const updateProductCategory = `-- name: UpdateProductCategory :one
UPDATE product_categories
SET name = $2, slug = $3, description = $4, sort_order = $5, active = $6, updated_at = now()
WHERE id = $1
RETURNING id, name, slug, description, sort_order, active, created_at, updated_at
`

type UpdateProductCategoryParams struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	SortOrder   int32     `json:"sort_order"`
	Active      bool      `json:"active"`
}

func (q *Queries) UpdateProductCategory(ctx context.Context, arg UpdateProductCategoryParams) (ProductCategory, error) {
	row := q.db.QueryRow(ctx, updateProductCategory, arg.ID, arg.Name, arg.Slug, arg.Description, arg.SortOrder, arg.Active)
	var i ProductCategory
	err := row.Scan(&i.ID, &i.Name, &i.Slug, &i.Description, &i.SortOrder, &i.Active, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const deleteProductCategory = `-- name: DeleteProductCategory :exec
DELETE FROM product_categories WHERE id = $1
`

func (q *Queries) DeleteProductCategory(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, deleteProductCategory, id)
	return err
}

// ================================ PRODUCTS ================================

const productCols = `id, category_id, sku, name, badge, price_vnd, cost_vnd, image_url, summary, description, spec_warranty, spec_trace, spec_delivery, spec_return, active, created_at, updated_at`

func scanProduct(row interface{ Scan(...any) error }) (Product, error) {
	var i Product
	err := row.Scan(
		&i.ID, &i.CategoryID, &i.Sku, &i.Name, &i.Badge, &i.PriceVnd, &i.CostVnd, &i.ImageUrl,
		&i.Summary, &i.Description, &i.SpecWarranty, &i.SpecTrace, &i.SpecDelivery, &i.SpecReturn,
		&i.Active, &i.CreatedAt, &i.UpdatedAt,
	)
	return i, err
}

const createProduct = `-- name: CreateProduct :one
INSERT INTO products (category_id, sku, name, badge, price_vnd, cost_vnd, image_url, summary, description, spec_warranty, spec_trace, spec_delivery, spec_return, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING ` + productCols

type CreateProductParams struct {
	CategoryID   uuid.NullUUID `json:"category_id"`
	Sku          string        `json:"sku"`
	Name         string        `json:"name"`
	Badge        string        `json:"badge"`
	PriceVnd     int64         `json:"price_vnd"`
	CostVnd      int64         `json:"cost_vnd"`
	ImageUrl     string        `json:"image_url"`
	Summary      string        `json:"summary"`
	Description  string        `json:"description"`
	SpecWarranty string        `json:"spec_warranty"`
	SpecTrace    string        `json:"spec_trace"`
	SpecDelivery string        `json:"spec_delivery"`
	SpecReturn   string        `json:"spec_return"`
	Active       bool          `json:"active"`
}

func (q *Queries) CreateProduct(ctx context.Context, arg CreateProductParams) (Product, error) {
	row := q.db.QueryRow(ctx, createProduct,
		arg.CategoryID, arg.Sku, arg.Name, arg.Badge, arg.PriceVnd, arg.CostVnd, arg.ImageUrl,
		arg.Summary, arg.Description, arg.SpecWarranty, arg.SpecTrace, arg.SpecDelivery, arg.SpecReturn, arg.Active,
	)
	return scanProduct(row)
}

const listProducts = `-- name: ListProducts :many
SELECT ` + productCols + ` FROM products ORDER BY created_at DESC
`

func (q *Queries) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := q.db.Query(ctx, listProducts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Product{}
	for rows.Next() {
		i, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listActiveProducts = `-- name: ListActiveProducts :many
SELECT ` + productCols + ` FROM products WHERE active = true ORDER BY name
`

func (q *Queries) ListActiveProducts(ctx context.Context) ([]Product, error) {
	rows, err := q.db.Query(ctx, listActiveProducts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Product{}
	for rows.Next() {
		i, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const getProduct = `-- name: GetProduct :one
SELECT ` + productCols + ` FROM products WHERE id = $1
`

func (q *Queries) GetProduct(ctx context.Context, id uuid.UUID) (Product, error) {
	row := q.db.QueryRow(ctx, getProduct, id)
	return scanProduct(row)
}

const updateProduct = `-- name: UpdateProduct :one
UPDATE products SET
    category_id = $2, sku = $3, name = $4, badge = $5, price_vnd = $6, cost_vnd = $7,
    image_url = $8, summary = $9, description = $10, spec_warranty = $11, spec_trace = $12,
    spec_delivery = $13, spec_return = $14, active = $15, updated_at = now()
WHERE id = $1
RETURNING ` + productCols

type UpdateProductParams struct {
	ID           uuid.UUID     `json:"id"`
	CategoryID   uuid.NullUUID `json:"category_id"`
	Sku          string        `json:"sku"`
	Name         string        `json:"name"`
	Badge        string        `json:"badge"`
	PriceVnd     int64         `json:"price_vnd"`
	CostVnd      int64         `json:"cost_vnd"`
	ImageUrl     string        `json:"image_url"`
	Summary      string        `json:"summary"`
	Description  string        `json:"description"`
	SpecWarranty string        `json:"spec_warranty"`
	SpecTrace    string        `json:"spec_trace"`
	SpecDelivery string        `json:"spec_delivery"`
	SpecReturn   string        `json:"spec_return"`
	Active       bool          `json:"active"`
}

func (q *Queries) UpdateProduct(ctx context.Context, arg UpdateProductParams) (Product, error) {
	row := q.db.QueryRow(ctx, updateProduct,
		arg.ID, arg.CategoryID, arg.Sku, arg.Name, arg.Badge, arg.PriceVnd, arg.CostVnd, arg.ImageUrl,
		arg.Summary, arg.Description, arg.SpecWarranty, arg.SpecTrace, arg.SpecDelivery, arg.SpecReturn, arg.Active,
	)
	return scanProduct(row)
}

const deleteProduct = `-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1
`

func (q *Queries) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, deleteProduct, id)
	return err
}
