// Hand-written to match sqlc output. Sales orders, items, distributions, commissions + saler stats.
package db

import (
	"context"

	"github.com/google/uuid"
)

const salesOrderCols = `id, code, customer_name, customer_phone, seller_id, affiliate_id, subtotal_vnd, cost_vnd, status, note, created_by, paid_by, paid_at, cancelled_at, created_at, updated_at`

func scanSalesOrder(row interface{ Scan(...any) error }) (SalesOrder, error) {
	var i SalesOrder
	err := row.Scan(
		&i.ID, &i.Code, &i.CustomerName, &i.CustomerPhone, &i.SellerID, &i.AffiliateID,
		&i.SubtotalVnd, &i.CostVnd, &i.Status, &i.Note, &i.CreatedBy, &i.PaidBy,
		&i.PaidAt, &i.CancelledAt, &i.CreatedAt, &i.UpdatedAt,
	)
	return i, err
}

const createSalesOrder = `-- name: CreateSalesOrder :one
INSERT INTO sales_orders (code, customer_name, customer_phone, seller_id, affiliate_id, subtotal_vnd, cost_vnd, note, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING ` + salesOrderCols

type CreateSalesOrderParams struct {
	Code          string        `json:"code"`
	CustomerName  string        `json:"customer_name"`
	CustomerPhone string        `json:"customer_phone"`
	SellerID      uuid.UUID     `json:"seller_id"`
	AffiliateID   uuid.NullUUID `json:"affiliate_id"`
	SubtotalVnd   int64         `json:"subtotal_vnd"`
	CostVnd       int64         `json:"cost_vnd"`
	Note          string        `json:"note"`
	CreatedBy     uuid.UUID     `json:"created_by"`
}

func (q *Queries) CreateSalesOrder(ctx context.Context, arg CreateSalesOrderParams) (SalesOrder, error) {
	row := q.db.QueryRow(ctx, createSalesOrder,
		arg.Code, arg.CustomerName, arg.CustomerPhone, arg.SellerID, arg.AffiliateID,
		arg.SubtotalVnd, arg.CostVnd, arg.Note, arg.CreatedBy,
	)
	return scanSalesOrder(row)
}

const getSalesOrder = `-- name: GetSalesOrder :one
SELECT ` + salesOrderCols + ` FROM sales_orders WHERE id = $1
`

func (q *Queries) GetSalesOrder(ctx context.Context, id uuid.UUID) (SalesOrder, error) {
	row := q.db.QueryRow(ctx, getSalesOrder, id)
	return scanSalesOrder(row)
}

const markSalesOrderPaid = `-- name: MarkSalesOrderPaid :one
UPDATE sales_orders SET status = 'paid', paid_by = $2, paid_at = now(), updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING ` + salesOrderCols

type MarkSalesOrderPaidParams struct {
	ID     uuid.UUID     `json:"id"`
	PaidBy uuid.NullUUID `json:"paid_by"`
}

func (q *Queries) MarkSalesOrderPaid(ctx context.Context, arg MarkSalesOrderPaidParams) (SalesOrder, error) {
	row := q.db.QueryRow(ctx, markSalesOrderPaid, arg.ID, arg.PaidBy)
	return scanSalesOrder(row)
}

const cancelSalesOrder = `-- name: CancelSalesOrder :one
UPDATE sales_orders SET status = 'cancelled', cancelled_at = now(), updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING ` + salesOrderCols

func (q *Queries) CancelSalesOrder(ctx context.Context, id uuid.UUID) (SalesOrder, error) {
	row := q.db.QueryRow(ctx, cancelSalesOrder, id)
	return scanSalesOrder(row)
}

// ListSalesOrdersRow joins seller/affiliate names for the admin order list.
type ListSalesOrdersRow struct {
	SalesOrder
	SellerName    string `json:"seller_name"`
	AffiliateName string `json:"affiliate_name"`
}

const listSalesOrders = `-- name: ListSalesOrders :many
SELECT o.id, o.code, o.customer_name, o.customer_phone, o.seller_id, o.affiliate_id, o.subtotal_vnd, o.cost_vnd, o.status, o.note, o.created_by, o.paid_by, o.paid_at, o.cancelled_at, o.created_at, o.updated_at,
       s.full_name AS seller_name, COALESCE(a.full_name, '') AS affiliate_name
FROM sales_orders o
JOIN users s ON s.id = o.seller_id
LEFT JOIN users a ON a.id = o.affiliate_id
ORDER BY o.created_at DESC
`

func scanListSalesOrdersRow(rows interface{ Scan(...any) error }) (ListSalesOrdersRow, error) {
	var i ListSalesOrdersRow
	err := rows.Scan(
		&i.ID, &i.Code, &i.CustomerName, &i.CustomerPhone, &i.SellerID, &i.AffiliateID,
		&i.SubtotalVnd, &i.CostVnd, &i.Status, &i.Note, &i.CreatedBy, &i.PaidBy,
		&i.PaidAt, &i.CancelledAt, &i.CreatedAt, &i.UpdatedAt,
		&i.SellerName, &i.AffiliateName,
	)
	return i, err
}

func (q *Queries) ListSalesOrders(ctx context.Context) ([]ListSalesOrdersRow, error) {
	rows, err := q.db.Query(ctx, listSalesOrders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListSalesOrdersRow{}
	for rows.Next() {
		i, err := scanListSalesOrdersRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listSalesOrdersBySeller = `-- name: ListSalesOrdersBySeller :many
SELECT o.id, o.code, o.customer_name, o.customer_phone, o.seller_id, o.affiliate_id, o.subtotal_vnd, o.cost_vnd, o.status, o.note, o.created_by, o.paid_by, o.paid_at, o.cancelled_at, o.created_at, o.updated_at,
       s.full_name AS seller_name, COALESCE(a.full_name, '') AS affiliate_name
FROM sales_orders o
JOIN users s ON s.id = o.seller_id
LEFT JOIN users a ON a.id = o.affiliate_id
WHERE o.seller_id = $1
ORDER BY o.created_at DESC
`

func (q *Queries) ListSalesOrdersBySeller(ctx context.Context, sellerID uuid.UUID) ([]ListSalesOrdersRow, error) {
	rows, err := q.db.Query(ctx, listSalesOrdersBySeller, sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListSalesOrdersRow{}
	for rows.Next() {
		i, err := scanListSalesOrdersRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listSalesOrdersByPhone = `-- name: ListSalesOrdersByPhone :many
SELECT ` + salesOrderCols + ` FROM sales_orders WHERE customer_phone = $1 ORDER BY created_at DESC
`

// ListSalesOrdersByPhone — đơn của 1 khách theo SĐT (dùng cho lịch sử đơn của khách đang đăng nhập).
func (q *Queries) ListSalesOrdersByPhone(ctx context.Context, customerPhone string) ([]SalesOrder, error) {
	rows, err := q.db.Query(ctx, listSalesOrdersByPhone, customerPhone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SalesOrder{}
	for rows.Next() {
		i, err := scanSalesOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ---- order items ----

const createSalesOrderItem = `-- name: CreateSalesOrderItem :one
INSERT INTO sales_order_items (order_id, product_id, name, qty, unit_price_vnd, unit_cost_vnd, line_total_vnd)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, order_id, product_id, name, qty, unit_price_vnd, unit_cost_vnd, line_total_vnd, created_at
`

type CreateSalesOrderItemParams struct {
	OrderID      uuid.UUID `json:"order_id"`
	ProductID    uuid.UUID `json:"product_id"`
	Name         string    `json:"name"`
	Qty          int64     `json:"qty"`
	UnitPriceVnd int64     `json:"unit_price_vnd"`
	UnitCostVnd  int64     `json:"unit_cost_vnd"`
	LineTotalVnd int64     `json:"line_total_vnd"`
}

func (q *Queries) CreateSalesOrderItem(ctx context.Context, arg CreateSalesOrderItemParams) (SalesOrderItem, error) {
	row := q.db.QueryRow(ctx, createSalesOrderItem,
		arg.OrderID, arg.ProductID, arg.Name, arg.Qty, arg.UnitPriceVnd, arg.UnitCostVnd, arg.LineTotalVnd)
	var i SalesOrderItem
	err := row.Scan(&i.ID, &i.OrderID, &i.ProductID, &i.Name, &i.Qty, &i.UnitPriceVnd, &i.UnitCostVnd, &i.LineTotalVnd, &i.CreatedAt)
	return i, err
}

const listSalesOrderItems = `-- name: ListSalesOrderItems :many
SELECT id, order_id, product_id, name, qty, unit_price_vnd, unit_cost_vnd, line_total_vnd, created_at
FROM sales_order_items WHERE order_id = $1 ORDER BY created_at
`

func (q *Queries) ListSalesOrderItems(ctx context.Context, orderID uuid.UUID) ([]SalesOrderItem, error) {
	rows, err := q.db.Query(ctx, listSalesOrderItems, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SalesOrderItem{}
	for rows.Next() {
		var i SalesOrderItem
		if err := rows.Scan(&i.ID, &i.OrderID, &i.ProductID, &i.Name, &i.Qty, &i.UnitPriceVnd, &i.UnitCostVnd, &i.LineTotalVnd, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ---- distribution (6-bucket breakdown) ----

const createSalesDistribution = `-- name: CreateSalesDistribution :one
INSERT INTO sales_distributions (order_id, total_vnd, seller_vnd, affiliate_vnd, equal_share_vnd, pool_vnd, cost_vnd, operations_vnd, dividend_pool_vnd)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING order_id, total_vnd, seller_vnd, affiliate_vnd, equal_share_vnd, pool_vnd, cost_vnd, operations_vnd, dividend_pool_vnd, swept, created_at
`

type CreateSalesDistributionParams struct {
	OrderID         uuid.UUID `json:"order_id"`
	TotalVnd        int64     `json:"total_vnd"`
	SellerVnd       int64     `json:"seller_vnd"`
	AffiliateVnd    int64     `json:"affiliate_vnd"`
	EqualShareVnd   int64     `json:"equal_share_vnd"`
	PoolVnd         int64     `json:"pool_vnd"`
	CostVnd         int64     `json:"cost_vnd"`
	OperationsVnd   int64     `json:"operations_vnd"`
	DividendPoolVnd int64     `json:"dividend_pool_vnd"`
}

func (q *Queries) CreateSalesDistribution(ctx context.Context, arg CreateSalesDistributionParams) (SalesDistribution, error) {
	row := q.db.QueryRow(ctx, createSalesDistribution,
		arg.OrderID, arg.TotalVnd, arg.SellerVnd, arg.AffiliateVnd, arg.EqualShareVnd,
		arg.PoolVnd, arg.CostVnd, arg.OperationsVnd, arg.DividendPoolVnd)
	var i SalesDistribution
	err := row.Scan(&i.OrderID, &i.TotalVnd, &i.SellerVnd, &i.AffiliateVnd, &i.EqualShareVnd,
		&i.PoolVnd, &i.CostVnd, &i.OperationsVnd, &i.DividendPoolVnd, &i.Swept, &i.CreatedAt)
	return i, err
}

const getSalesDistribution = `-- name: GetSalesDistribution :one
SELECT order_id, total_vnd, seller_vnd, affiliate_vnd, equal_share_vnd, pool_vnd, cost_vnd, operations_vnd, dividend_pool_vnd, swept, created_at
FROM sales_distributions WHERE order_id = $1
`

func (q *Queries) GetSalesDistribution(ctx context.Context, orderID uuid.UUID) (SalesDistribution, error) {
	row := q.db.QueryRow(ctx, getSalesDistribution, orderID)
	var i SalesDistribution
	err := row.Scan(&i.OrderID, &i.TotalVnd, &i.SellerVnd, &i.AffiliateVnd, &i.EqualShareVnd,
		&i.PoolVnd, &i.CostVnd, &i.OperationsVnd, &i.DividendPoolVnd, &i.Swept, &i.CreatedAt)
	return i, err
}

// ---- dividend sweep (đơn paid chưa gộp cổ tức) ----

// UnsweptDistributionRow là phần tối thiểu để quét cổ tức: khoản pool cổ đông (15%) đã trích sẵn
// mỗi đơn cùng doanh thu gốc, cho các đơn ĐÃ paid mà CHƯA swept.
type UnsweptDistributionRow struct {
	OrderID         uuid.UUID `json:"order_id"`
	TotalVnd        int64     `json:"total_vnd"`
	DividendPoolVnd int64     `json:"dividend_pool_vnd"`
}

const listUnsweptPaidDistributions = `-- name: ListUnsweptPaidDistributions :many
SELECT sd.order_id, sd.total_vnd, sd.dividend_pool_vnd
FROM sales_distributions sd
JOIN sales_orders so ON so.id = sd.order_id
WHERE sd.swept = false AND so.status = 'paid'
ORDER BY sd.created_at
FOR UPDATE OF sd
`

// ListUnsweptPaidDistributions khoá (FOR UPDATE) các dòng phân bổ chưa gộp của đơn paid — nền tảng
// idempotent của Sweep: hai lần quét đồng thời, lần sau chờ khoá rồi thấy rỗng (đã swept).
func (q *Queries) ListUnsweptPaidDistributions(ctx context.Context) ([]UnsweptDistributionRow, error) {
	rows, err := q.db.Query(ctx, listUnsweptPaidDistributions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []UnsweptDistributionRow{}
	for rows.Next() {
		var i UnsweptDistributionRow
		if err := rows.Scan(&i.OrderID, &i.TotalVnd, &i.DividendPoolVnd); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const sumUnsweptPaidPool = `-- name: SumUnsweptPaidPool :one
SELECT COUNT(*)::bigint AS orders,
       COALESCE(SUM(sd.total_vnd), 0)::bigint AS revenue,
       COALESCE(SUM(sd.dividend_pool_vnd), 0)::bigint AS pool
FROM sales_distributions sd
JOIN sales_orders so ON so.id = sd.order_id
WHERE sd.swept = false AND so.status = 'paid'
`

// SumUnsweptPaidPoolRow tổng hợp KHÔNG khoá — để preview số sẽ quét mà không đụng ghi.
type SumUnsweptPaidPoolRow struct {
	Orders  int64 `json:"orders"`
	Revenue int64 `json:"revenue"`
	Pool    int64 `json:"pool"`
}

func (q *Queries) SumUnsweptPaidPool(ctx context.Context) (SumUnsweptPaidPoolRow, error) {
	row := q.db.QueryRow(ctx, sumUnsweptPaidPool)
	var i SumUnsweptPaidPoolRow
	err := row.Scan(&i.Orders, &i.Revenue, &i.Pool)
	return i, err
}

const markSalesDistributionSwept = `-- name: MarkSalesDistributionSwept :execrows
UPDATE sales_distributions SET swept = true WHERE order_id = $1 AND swept = false
`

// MarkSalesDistributionSwept đánh dấu 1 đơn đã gộp cổ tức. Trả số dòng đổi (0 nếu đã swept trước đó).
func (q *Queries) MarkSalesDistributionSwept(ctx context.Context, orderID uuid.UUID) (int64, error) {
	tag, err := q.db.Exec(ctx, markSalesDistributionSwept, orderID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---- sales commissions ----

const salesCommissionCols = `id, order_id, beneficiary_id, kind, base_amount, rate, amount, tax_pit, net_amount, status, approved_by, paid_at, created_at`

func scanSalesCommission(row interface{ Scan(...any) error }) (SalesCommission, error) {
	var i SalesCommission
	err := row.Scan(&i.ID, &i.OrderID, &i.BeneficiaryID, &i.Kind, &i.BaseAmount, &i.Rate,
		&i.Amount, &i.TaxPit, &i.NetAmount, &i.Status, &i.ApprovedBy, &i.PaidAt, &i.CreatedAt)
	return i, err
}

const createSalesCommission = `-- name: CreateSalesCommission :one
INSERT INTO sales_commissions (order_id, beneficiary_id, kind, base_amount, rate, amount, tax_pit, net_amount, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')
ON CONFLICT (order_id, kind) DO NOTHING
RETURNING ` + salesCommissionCols

type CreateSalesCommissionParams struct {
	OrderID       uuid.UUID           `json:"order_id"`
	BeneficiaryID uuid.UUID           `json:"beneficiary_id"`
	Kind          SalesCommissionKind `json:"kind"`
	BaseAmount    int64               `json:"base_amount"`
	Rate          float64             `json:"rate"`
	Amount        int64               `json:"amount"`
	TaxPit        int64               `json:"tax_pit"`
	NetAmount     int64               `json:"net_amount"`
}

func (q *Queries) CreateSalesCommission(ctx context.Context, arg CreateSalesCommissionParams) (SalesCommission, error) {
	row := q.db.QueryRow(ctx, createSalesCommission,
		arg.OrderID, arg.BeneficiaryID, arg.Kind, arg.BaseAmount, arg.Rate, arg.Amount, arg.TaxPit, arg.NetAmount)
	return scanSalesCommission(row)
}

const listSalesCommissionsByBeneficiary = `-- name: ListSalesCommissionsByBeneficiary :many
SELECT ` + salesCommissionCols + ` FROM sales_commissions WHERE beneficiary_id = $1 ORDER BY created_at DESC
`

func (q *Queries) ListSalesCommissionsByBeneficiary(ctx context.Context, beneficiaryID uuid.UUID) ([]SalesCommission, error) {
	rows, err := q.db.Query(ctx, listSalesCommissionsByBeneficiary, beneficiaryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SalesCommission{}
	for rows.Next() {
		i, err := scanSalesCommission(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const sumSalesCommissionEarnedByBeneficiary = `-- name: SumSalesCommissionEarnedByBeneficiary :one
SELECT COALESCE(SUM(net_amount), 0)::bigint AS total
FROM sales_commissions WHERE beneficiary_id = $1 AND status <> 'rejected'
`

func (q *Queries) SumSalesCommissionEarnedByBeneficiary(ctx context.Context, beneficiaryID uuid.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, sumSalesCommissionEarnedByBeneficiary, beneficiaryID)
	var total int64
	err := row.Scan(&total)
	return total, err
}

const sumPendingSalesCommissionByBeneficiary = `-- name: SumPendingSalesCommissionByBeneficiary :one
SELECT COALESCE(SUM(net_amount), 0)::bigint AS total
FROM sales_commissions WHERE beneficiary_id = $1 AND status = 'pending'
`

// Hoa hồng bán hàng (net) CHƯA DUYỆT của 1 người — đang chờ admin duyệt.
func (q *Queries) SumPendingSalesCommissionByBeneficiary(ctx context.Context, beneficiaryID uuid.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, sumPendingSalesCommissionByBeneficiary, beneficiaryID)
	var total int64
	err := row.Scan(&total)
	return total, err
}

// ---- saler monitoring (admin) ----

type SalerStatsRow struct {
	SellerID      uuid.UUID `json:"seller_id"`
	FullName      string    `json:"full_name"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	PaidOrders    int64     `json:"paid_orders"`
	PendingOrders int64     `json:"pending_orders"`
	RevenueVnd    int64     `json:"revenue_vnd"`
	CommissionNet int64     `json:"commission_net_vnd"`
}

const salerStats = `-- name: SalerStats :many
SELECT u.id, u.full_name, u.email, u.phone,
  COUNT(o.id) FILTER (WHERE o.status = 'paid')    AS paid_orders,
  COUNT(o.id) FILTER (WHERE o.status = 'pending') AS pending_orders,
  COALESCE(SUM(o.subtotal_vnd) FILTER (WHERE o.status = 'paid'), 0)::bigint AS revenue,
  COALESCE((SELECT SUM(sc.net_amount) FROM sales_commissions sc WHERE sc.beneficiary_id = u.id AND sc.status <> 'rejected'), 0)::bigint AS commission_net
FROM users u
LEFT JOIN sales_orders o ON o.seller_id = u.id
WHERE u.role = 'saler'
GROUP BY u.id, u.full_name, u.email, u.phone
ORDER BY revenue DESC, u.full_name
`

func (q *Queries) SalerStats(ctx context.Context) ([]SalerStatsRow, error) {
	rows, err := q.db.Query(ctx, salerStats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SalerStatsRow{}
	for rows.Next() {
		var i SalerStatsRow
		if err := rows.Scan(&i.SellerID, &i.FullName, &i.Email, &i.Phone,
			&i.PaidOrders, &i.PendingOrders, &i.RevenueVnd, &i.CommissionNet); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listSalers = `-- name: ListSalers :many
SELECT id, full_name, email, phone FROM users WHERE role = 'saler' ORDER BY full_name
`

type ListSalersRow struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
	Phone    string    `json:"phone"`
}

func (q *Queries) ListSalers(ctx context.Context) ([]ListSalersRow, error) {
	rows, err := q.db.Query(ctx, listSalers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListSalersRow{}
	for rows.Next() {
		var i ListSalersRow
		if err := rows.Scan(&i.ID, &i.FullName, &i.Email, &i.Phone); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const deleteSalesOrder = `DELETE FROM sales_orders WHERE id = $1`

// DeleteSalesOrder xoá đơn (CASCADE items/distributions/commissions theo FK).
func (q *Queries) DeleteSalesOrder(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, deleteSalesOrder, id)
	return err
}
