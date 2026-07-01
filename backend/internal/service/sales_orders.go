package service

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/idgen"
)

// ----------------------------- Đơn hàng -----------------------------

type OrderItemInput struct {
	ProductID string `json:"product_id"`
	Qty       int64  `json:"qty"`
}

type OrderInput struct {
	CustomerName  string           `json:"customer_name"`
	CustomerPhone string           `json:"customer_phone"`
	SellerID      string           `json:"seller_id"`    // admin chọn; saler bỏ qua (tự là người bán)
	AffiliateID   string           `json:"affiliate_id"` // tuỳ chọn
	Note          string           `json:"note"`
	Items         []OrderItemInput `json:"items"`
}

// OrderDetail gộp đơn + dòng hàng + breakdown chia tiền (nếu đã paid).
type OrderDetail struct {
	Order        db.SalesOrder         `json:"order"`
	Items        []db.SalesOrderItem   `json:"items"`
	Distribution *db.SalesDistribution `json:"distribution,omitempty"`
	SellerName   string                `json:"seller_name"`
	AffiliateName string               `json:"affiliate_name"`
}

// CreateOrder: saler tạo đơn cho chính mình (seller = actor) HOẶC admin nhập đơn và chọn seller.
func (s *SalesService) CreateOrder(ctx context.Context, actor uuid.UUID, isAdmin bool, in OrderInput) (db.SalesOrder, error) {
	if len(in.Items) == 0 {
		return db.SalesOrder{}, errors.Join(ErrValidation, errors.New("đơn hàng phải có ít nhất 1 sản phẩm"))
	}

	// Người bán: admin chọn seller_id; saler thì luôn là chính mình.
	sellerID := actor
	if isAdmin && in.SellerID != "" {
		id, err := uuid.Parse(in.SellerID)
		if err != nil {
			return db.SalesOrder{}, ErrValidation
		}
		sellerID = id
	}
	// Người bán phải là tài khoản saler (bán hàng tách biệt với đầu tư).
	seller, err := s.store.GetUserByID(ctx, sellerID)
	if err != nil {
		return db.SalesOrder{}, errors.Join(ErrValidation, errors.New("người bán không hợp lệ"))
	}
	if seller.Role != db.UserRoleSaler {
		return db.SalesOrder{}, errors.Join(ErrValidation, errors.New("người bán phải là tài khoản bán hàng (saler)"))
	}

	var affiliate uuid.NullUUID
	if in.AffiliateID != "" {
		id, err := uuid.Parse(in.AffiliateID)
		if err != nil {
			return db.SalesOrder{}, ErrValidation
		}
		if id == sellerID {
			return db.SalesOrder{}, errors.Join(ErrValidation, errors.New("affiliate không thể trùng người bán"))
		}
		affiliate = uuid.NullUUID{UUID: id, Valid: true}
	}

	var order db.SalesOrder
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		// Tính subtotal + giá vốn từ sản phẩm (snapshot giá lúc tạo đơn).
		var subtotal, cost int64
		type line struct {
			p   db.Product
			qty int64
		}
		lines := make([]line, 0, len(in.Items))
		for _, it := range in.Items {
			pid, e := uuid.Parse(it.ProductID)
			if e != nil || it.Qty <= 0 {
				return ErrValidation
			}
			p, e := q.GetProduct(ctx, pid)
			if errors.Is(e, pgx.ErrNoRows) {
				return errors.Join(ErrValidation, errors.New("sản phẩm không tồn tại"))
			}
			if e != nil {
				return e
			}
			subtotal += p.PriceVnd * it.Qty
			cost += p.CostVnd * it.Qty
			lines = append(lines, line{p: p, qty: it.Qty})
		}
		if subtotal <= 0 {
			return ErrValidation
		}

		o, e := q.CreateSalesOrder(ctx, db.CreateSalesOrderParams{
			Code: idgen.SalesOrderCode(), CustomerName: in.CustomerName, CustomerPhone: in.CustomerPhone,
			SellerID: sellerID, AffiliateID: affiliate, SubtotalVnd: subtotal, CostVnd: cost,
			Note: in.Note, CreatedBy: actor,
		})
		if e != nil {
			return e
		}
		for _, ln := range lines {
			if _, e := q.CreateSalesOrderItem(ctx, db.CreateSalesOrderItemParams{
				OrderID: o.ID, ProductID: ln.p.ID, Name: ln.p.Name, Qty: ln.qty,
				UnitPriceVnd: ln.p.PriceVnd, UnitCostVnd: ln.p.CostVnd, LineTotalVnd: ln.p.PriceVnd * ln.qty,
			}); e != nil {
				return e
			}
		}
		order = o
		return audit.Write(ctx, q, audit.Actor(actor), "sales_order.create", "sales_orders", o.ID.String(), nil, o)
	})
	return order, err
}

// salesRates gom 6 tỷ lệ chia + ngưỡng đồng chia (đọc từ settings, có default = chính sách 25/10/5/15/30/15).
type salesRates struct {
	seller, affiliate, equalShare, pool, cost, operations float64
	equalShareMin                                          int64
}

func (s *SalesService) rates(ctx context.Context) salesRates {
	f := func(k string, d float64) float64 { return s.settings.Float(ctx, k, d) }
	return salesRates{
		seller:        f("sales_seller_rate", 0.25),
		affiliate:     f("sales_affiliate_rate", 0.10),
		equalShare:    f("sales_equalshare_rate", 0.05),
		pool:          f("sales_pool_rate", 0.15),
		cost:          f("sales_cost_rate", 0.30),
		operations:    f("sales_operations_rate", 0.15),
		equalShareMin: int64(f("sales_equalshare_min", 1_000_000)),
	}
}

// splitOrder chia subtotal thành 6 khoản đúng công thức. Phần lẻ do làm tròn + các khoản không áp
// dụng (affiliate khi không có người giới thiệu; đồng chia khi đơn < ngưỡng) dồn vào VẬN HÀNH để
// tổng 6 khoản LUÔN = subtotal (không thất thoát/đẻ thêm đồng nào).
//
// LƯU Ý cơ chế: 15% pool_vnd = Pool Cổ Đông (chia cho CỔ ĐÔNG theo 9% đồng chia + 6% bonus hạng).
// 5% equal_share_vnd = pool ĐỒNG CHIA RIÊNG cho MỌI người có đơn ≥1tr (người mua), TÁCH BIỆT, KHÔNG
// thuộc pool cổ đông. Vì vậy DividendPoolVnd (phần vào pool cổ đông) = pool_vnd (15%) MÀ THÔI.
func splitOrder(subtotal int64, hasAffiliate bool, r salesRates) db.CreateSalesDistributionParams {
	round := func(rate float64) int64 { return int64(math.Round(float64(subtotal) * rate)) }
	seller := round(r.seller)
	affiliate := int64(0)
	if hasAffiliate {
		affiliate = round(r.affiliate)
	}
	equalShare := int64(0)
	if subtotal >= r.equalShareMin {
		equalShare = round(r.equalShare)
	}
	pool := round(r.pool)
	cost := round(r.cost)
	operations := subtotal - (seller + affiliate + equalShare + pool + cost) // hấp thụ phần còn lại
	return db.CreateSalesDistributionParams{
		TotalVnd: subtotal, SellerVnd: seller, AffiliateVnd: affiliate, EqualShareVnd: equalShare,
		PoolVnd: pool, CostVnd: cost, OperationsVnd: operations, DividendPoolVnd: pool,
	}
}

// PayOrder xác nhận thanh toán → chia 6 khoản, sinh hoa hồng người bán + affiliate (−10% TNCN) vào
// ví chung, ghi breakdown. Tất cả trong 1 giao dịch. Actor = admin hoặc chính người bán của đơn.
func (s *SalesService) PayOrder(ctx context.Context, actor uuid.UUID, isAdmin bool, orderID uuid.UUID) (OrderDetail, error) {
	r := s.rates(ctx)
	pit := s.settings.Float(ctx, "pit_rate", 0.10)
	var detail OrderDetail
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		o, e := q.GetSalesOrder(ctx, orderID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if !isAdmin && o.SellerID != actor {
			return ErrForbidden
		}
		if o.Status != db.SalesOrderStatusPending {
			return ErrInvalidState
		}

		dist := splitOrder(o.SubtotalVnd, o.AffiliateID.Valid, r)
		dist.OrderID = o.ID
		if _, e := q.CreateSalesDistribution(ctx, dist); e != nil {
			return e
		}

		// Hoa hồng người bán (25%) − 10% TNCN → ví chung.
		if e := s.createSalesCommission(ctx, q, o.ID, o.SellerID, db.SalesCommissionKindSeller, o.SubtotalVnd, dist.SellerVnd, r.seller, pit); e != nil {
			return e
		}
		// Hoa hồng affiliate (10%) nếu có.
		if o.AffiliateID.Valid && dist.AffiliateVnd > 0 {
			if e := s.createSalesCommission(ctx, q, o.ID, o.AffiliateID.UUID, db.SalesCommissionKindAffiliate, o.SubtotalVnd, dist.AffiliateVnd, r.affiliate, pit); e != nil {
				return e
			}
		}

		paid, e := q.MarkSalesOrderPaid(ctx, db.MarkSalesOrderPaidParams{ID: o.ID, PaidBy: uuid.NullUUID{UUID: actor, Valid: true}})
		if e != nil {
			return e
		}
		detail.Order = paid
		return audit.Write(ctx, q, audit.Actor(actor), "sales_order.pay", "sales_orders", o.ID.String(), o, paid)
	})
	if err != nil {
		return OrderDetail{}, err
	}
	return s.OrderDetail(ctx, orderID)
}

func (s *SalesService) createSalesCommission(ctx context.Context, q *db.Queries, orderID, beneficiary uuid.UUID, kind db.SalesCommissionKind, base, gross int64, rate, pit float64) error {
	if gross <= 0 {
		return nil
	}
	tax := int64(math.Round(float64(gross) * pit))
	_, err := q.CreateSalesCommission(ctx, db.CreateSalesCommissionParams{
		OrderID: orderID, BeneficiaryID: beneficiary, Kind: kind, BaseAmount: base,
		Rate: rate, Amount: gross, TaxPit: tax, NetAmount: gross - tax,
	})
	return err
}

func (s *SalesService) CancelOrder(ctx context.Context, actor uuid.UUID, isAdmin bool, orderID uuid.UUID) (db.SalesOrder, error) {
	var out db.SalesOrder
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		o, e := q.GetSalesOrder(ctx, orderID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if !isAdmin && o.SellerID != actor {
			return ErrForbidden
		}
		out, e = q.CancelSalesOrder(ctx, orderID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidState // chỉ huỷ được đơn đang chờ
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(actor), "sales_order.cancel", "sales_orders", orderID.String(), o, out)
	})
	return out, err
}

func (s *SalesService) OrderDetail(ctx context.Context, orderID uuid.UUID) (OrderDetail, error) {
	o, err := s.store.GetSalesOrder(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrderDetail{}, ErrNotFound
	}
	if err != nil {
		return OrderDetail{}, err
	}
	items, err := s.store.ListSalesOrderItems(ctx, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	d := OrderDetail{Order: o, Items: items}
	if seller, e := s.store.GetUserByID(ctx, o.SellerID); e == nil {
		d.SellerName = seller.FullName
	}
	if o.AffiliateID.Valid {
		if aff, e := s.store.GetUserByID(ctx, o.AffiliateID.UUID); e == nil {
			d.AffiliateName = aff.FullName
		}
	}
	if dist, e := s.store.GetSalesDistribution(ctx, orderID); e == nil {
		d.Distribution = &dist
	}
	return d, nil
}

func (s *SalesService) ListOrders(ctx context.Context) ([]db.ListSalesOrdersRow, error) {
	return s.store.ListSalesOrders(ctx)
}

func (s *SalesService) ListOrdersBySeller(ctx context.Context, sellerID uuid.UUID) ([]db.ListSalesOrdersRow, error) {
	return s.store.ListSalesOrdersBySeller(ctx, sellerID)
}

// ----------------------------- Giám sát saler -----------------------------

func (s *SalesService) SalerStats(ctx context.Context) ([]db.SalerStatsRow, error) {
	return s.store.SalerStats(ctx)
}

func (s *SalesService) ListSalers(ctx context.Context) ([]db.ListSalersRow, error) {
	return s.store.ListSalers(ctx)
}

func (s *SalesService) ListMyCommissions(ctx context.Context, beneficiary uuid.UUID) ([]db.SalesCommission, error) {
	return s.store.ListSalesCommissionsByBeneficiary(ctx, beneficiary)
}
