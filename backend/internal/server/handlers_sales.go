package server

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

func isAdmin(r *http.Request) bool { return userRole(r) == "admin" }

// ----------------------------- Sản phẩm (CÔNG KHAI) -----------------------------
// Website bán hàng duoclieuhk.vn đọc catalog qua 2 route công khai này (không auth,
// chỉ trả hàng đang bán). Nguồn dữ liệu duy nhất vẫn là admin CMS ở HK SHAREHOLDER.

// publicProduct: DTO công khai — CỐ Ý loại bỏ cost_vnd (giá vốn/biên lợi nhuận nội bộ),
// category_id, active và timestamp. Chỉ lộ những gì cần cho trang bán hàng.
type publicProduct struct {
	ID           string `json:"id"`
	Sku          string `json:"sku"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Badge        string `json:"badge"`
	PriceVnd     int64  `json:"price_vnd"`
	ImageURL     string `json:"image_url"`
	Summary      string `json:"summary"`
	Description  string `json:"description"`
	SpecWarranty string `json:"spec_warranty"`
	SpecTrace    string `json:"spec_trace"`
	SpecDelivery string `json:"spec_delivery"`
	SpecReturn   string `json:"spec_return"`
}

func toPublicProduct(p db.Product) publicProduct {
	return publicProduct{
		ID: p.ID.String(), Sku: p.Sku, Name: p.Name, Slug: p.Slug, Badge: p.Badge,
		PriceVnd: p.PriceVnd, ImageURL: p.ImageUrl, Summary: p.Summary, Description: p.Description,
		SpecWarranty: p.SpecWarranty, SpecTrace: p.SpecTrace,
		SpecDelivery: p.SpecDelivery, SpecReturn: p.SpecReturn,
	}
}

func (s *Server) handlePublicProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.sales.ListActiveProducts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]publicProduct, 0, len(products))
	for _, p := range products {
		out = append(out, toPublicProduct(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": out})
}

// handlePublicCheckout: khách đặt hàng online từ giỏ hàng (không cần đăng nhập).
func (s *Server) handlePublicCheckout(w http.ResponseWriter, r *http.Request) {
	var in service.PublicCheckoutInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	res, err := s.sales.PublicCheckout(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handlePublicProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, service.ErrValidation)
		return
	}
	p, err := s.sales.GetActiveProductBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPublicProduct(p))
}

// ----------------------------- Danh mục (admin) -----------------------------

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.sales.ListCategories(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var in service.CategoryInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.sales.CreateCategory(r.Context(), userID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.CategoryInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.sales.UpdateCategory(r.Context(), userID(r), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.sales.DeleteCategory(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ----------------------------- Sản phẩm (admin) -----------------------------

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.sales.ListProducts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var in service.ProductInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	p, err := s.sales.CreateProduct(r.Context(), userID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.ProductInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	p, err := s.sales.UpdateProduct(r.Context(), userID(r), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.sales.DeleteProduct(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleActiveProducts: danh sách sản phẩm đang bán (saler dùng để tạo đơn). Trả full list,
// FE tự lọc active nếu cần — nhưng saler chỉ nên thấy hàng đang bán.
func (s *Server) handleActiveProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.sales.ListProducts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

// ----------------------------- Upload ảnh sản phẩm -----------------------------

// POST /api/v1/admin/products/image — admin upload ảnh sản phẩm (PUBLIC, không mã hoá).
func (s *Server) handleUploadProductImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 13<<20))
	if err != nil {
		writeError(w, err)
		return
	}
	up, err := s.upload.SaveProductImage(r.Context(), userID(r), hdr.Header.Get("Content-Type"), data)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":  up.ID.String(),
		"url": "/api/v1/public/images/" + up.ID.String(),
	})
}

// GET /api/v1/public/images/{id} — phục vụ ảnh sản phẩm công khai (no auth).
func (s *Server) handleGetPublicImage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	data, contentType, err := s.upload.LoadPublicImage(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// ----------------------------- Đơn hàng -----------------------------

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var in service.OrderInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	o, err := s.sales.CreateOrder(r.Context(), userID(r), isAdmin(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) handlePayOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	d, err := s.sales.PayOrder(r.Context(), userID(r), isAdmin(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	o, err := s.sales.CancelOrder(r.Context(), userID(r), isAdmin(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) handleOrderDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	d, err := s.sales.OrderDetail(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	// Saler chỉ xem được đơn của chính mình.
	if !isAdmin(r) && d.Order.SellerID != userID(r) {
		writeError(w, service.ErrForbidden)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// Admin: tất cả đơn. Saler: chỉ đơn của mình.
func (s *Server) handleListAllOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.sales.ListOrders(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleMyOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.sales.ListOrdersBySeller(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// handleMyCustomerOrders — lịch sử đơn MUA của người đang đăng nhập (khớp theo SĐT hồ sơ).
func (s *Server) handleMyCustomerOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.sales.ListMyCustomerOrders(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// ----------------------------- Giám sát saler -----------------------------

func (s *Server) handleSalerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.sales.SalerStats(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleListSalers(w http.ResponseWriter, r *http.Request) {
	salers, err := s.sales.ListSalers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, salers)
}

func (s *Server) handleMySalesCommissions(w http.ResponseWriter, r *http.Request) {
	cs, err := s.sales.ListMyCommissions(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

// handleDeleteOrder — admin xoá đơn bán.
func (s *Server) handleDeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.sales.DeleteOrder(r.Context(), userID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
