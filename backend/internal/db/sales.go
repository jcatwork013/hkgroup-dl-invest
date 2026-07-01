// Hand-written to match sqlc output (sqlc generate is blocked by legacy reservation.sql).
// Models + enums for the sales module (migration 00028).
package db

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// New role for sales accounts (separate from investor/admin).
const UserRoleSaler UserRole = "saler"

// ---- sales_order_status enum ----
type SalesOrderStatus string

const (
	SalesOrderStatusPending   SalesOrderStatus = "pending"
	SalesOrderStatusPaid      SalesOrderStatus = "paid"
	SalesOrderStatusCancelled SalesOrderStatus = "cancelled"
)

func (e *SalesOrderStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = SalesOrderStatus(s)
	case string:
		*e = SalesOrderStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for SalesOrderStatus: %T", src)
	}
	return nil
}

type NullSalesOrderStatus struct {
	SalesOrderStatus SalesOrderStatus `json:"sales_order_status"`
	Valid            bool             `json:"valid"`
}

func (ns *NullSalesOrderStatus) Scan(value interface{}) error {
	if value == nil {
		ns.SalesOrderStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.SalesOrderStatus.Scan(value)
}

func (ns NullSalesOrderStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.SalesOrderStatus), nil
}

// ---- sales_commission_kind enum ----
type SalesCommissionKind string

const (
	SalesCommissionKindSeller    SalesCommissionKind = "seller"
	SalesCommissionKindAffiliate SalesCommissionKind = "affiliate"
)

func (e *SalesCommissionKind) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = SalesCommissionKind(s)
	case string:
		*e = SalesCommissionKind(s)
	default:
		return fmt.Errorf("unsupported scan type for SalesCommissionKind: %T", src)
	}
	return nil
}

// ---- table models ----
type ProductCategory struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Description string             `json:"description"`
	SortOrder   int32              `json:"sort_order"`
	Active      bool               `json:"active"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
	UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
}

type Product struct {
	ID           uuid.UUID          `json:"id"`
	CategoryID   uuid.NullUUID      `json:"category_id"`
	Sku          string             `json:"sku"`
	Name         string             `json:"name"`
	Slug         string             `json:"slug"`
	Badge        string             `json:"badge"`
	PriceVnd     int64              `json:"price_vnd"`
	CostVnd      int64              `json:"cost_vnd"`
	ImageUrl     string             `json:"image_url"`
	Summary      string             `json:"summary"`
	Description  string             `json:"description"`
	SpecWarranty string             `json:"spec_warranty"`
	SpecTrace    string             `json:"spec_trace"`
	SpecDelivery string             `json:"spec_delivery"`
	SpecReturn   string             `json:"spec_return"`
	Active       bool               `json:"active"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz `json:"updated_at"`
}

type SalesOrder struct {
	ID            uuid.UUID          `json:"id"`
	Code          string             `json:"code"`
	CustomerName  string             `json:"customer_name"`
	CustomerPhone string             `json:"customer_phone"`
	SellerID      uuid.UUID          `json:"seller_id"`
	AffiliateID   uuid.NullUUID      `json:"affiliate_id"`
	SubtotalVnd   int64              `json:"subtotal_vnd"`
	CostVnd       int64              `json:"cost_vnd"`
	Status        SalesOrderStatus   `json:"status"`
	Note          string             `json:"note"`
	CreatedBy     uuid.UUID          `json:"created_by"`
	PaidBy        uuid.NullUUID      `json:"paid_by"`
	PaidAt        pgtype.Timestamptz `json:"paid_at"`
	CancelledAt   pgtype.Timestamptz `json:"cancelled_at"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
	UpdatedAt     pgtype.Timestamptz `json:"updated_at"`
}

type SalesOrderItem struct {
	ID           uuid.UUID          `json:"id"`
	OrderID      uuid.UUID          `json:"order_id"`
	ProductID    uuid.UUID          `json:"product_id"`
	Name         string             `json:"name"`
	Qty          int64              `json:"qty"`
	UnitPriceVnd int64              `json:"unit_price_vnd"`
	UnitCostVnd  int64              `json:"unit_cost_vnd"`
	LineTotalVnd int64              `json:"line_total_vnd"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
}

type SalesDistribution struct {
	OrderID         uuid.UUID          `json:"order_id"`
	TotalVnd        int64              `json:"total_vnd"`
	SellerVnd       int64              `json:"seller_vnd"`
	AffiliateVnd    int64              `json:"affiliate_vnd"`
	EqualShareVnd   int64              `json:"equal_share_vnd"`
	PoolVnd         int64              `json:"pool_vnd"`
	CostVnd         int64              `json:"cost_vnd"`
	OperationsVnd   int64              `json:"operations_vnd"`
	DividendPoolVnd int64              `json:"dividend_pool_vnd"`
	Swept           bool               `json:"swept"`
	CreatedAt       pgtype.Timestamptz `json:"created_at"`
}

type SalesCommission struct {
	ID            uuid.UUID           `json:"id"`
	OrderID       uuid.UUID           `json:"order_id"`
	BeneficiaryID uuid.UUID           `json:"beneficiary_id"`
	Kind          SalesCommissionKind `json:"kind"`
	BaseAmount    int64               `json:"base_amount"`
	Rate          float64             `json:"rate"`
	Amount        int64               `json:"amount"`
	TaxPit        int64               `json:"tax_pit"`
	NetAmount     int64               `json:"net_amount"`
	Status        CommissionStatus    `json:"status"`
	ApprovedBy    uuid.NullUUID       `json:"approved_by"`
	PaidAt        pgtype.Timestamptz  `json:"paid_at"`
	CreatedAt     pgtype.Timestamptz  `json:"created_at"`
}
