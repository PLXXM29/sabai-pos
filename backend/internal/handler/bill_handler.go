package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"minimart-pos/backend/internal/domain"
	"minimart-pos/backend/internal/middleware"
	"minimart-pos/backend/internal/receipt"
	"minimart-pos/backend/internal/service"
)

type BillHandler struct {
	sale *service.SaleService
	log  *zap.Logger
}

func NewBillHandler(sale *service.SaleService, log *zap.Logger) *BillHandler {
	return &BillHandler{sale: sale, log: log}
}

type checkoutLineReq struct {
	ProductID string `json:"product_id"`
	Qty       int32  `json:"qty"`
}

type checkoutReq struct {
	Lines         []checkoutLineReq `json:"lines"`
	PaymentMethod string            `json:"payment_method"`
	Paid          int64             `json:"paid"`     // satang
	Discount      int64             `json:"discount"` // satang
	ClientUUID    *string           `json:"client_uuid"`
}

func (h *BillHandler) Checkout(c *gin.Context) {
	var req checkoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	lines := make([]service.CheckoutLine, 0, len(req.Lines))
	for _, l := range req.Lines {
		pid, err := uuid.Parse(l.ProductID)
		if err != nil {
			writeError(c, h.log, domain.Validation("product_id ไม่ถูกต้อง"))
			return
		}
		lines = append(lines, service.CheckoutLine{ProductID: pid, Qty: l.Qty})
	}
	var clientUUID *uuid.UUID
	if req.ClientUUID != nil && *req.ClientUUID != "" {
		parsed, err := uuid.Parse(*req.ClientUUID)
		if err != nil {
			writeError(c, h.log, domain.Validation("client_uuid ไม่ถูกต้อง"))
			return
		}
		clientUUID = &parsed
	}
	d, err := h.sale.Checkout(c.Request.Context(), middleware.StoreID(c), service.CheckoutInput{
		Lines:         lines,
		PaymentMethod: req.PaymentMethod,
		Paid:          req.Paid,
		Discount:      req.Discount,
		ClientUUID:    clientUUID,
	}, middleware.UserID(c))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, billResponse(d))
}

func (h *BillHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	d, err := h.sale.GetBillDetail(c.Request.Context(), middleware.StoreID(c), id)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, billResponse(d))
}

type voidReq struct {
	Reason string `json:"reason"`
}

func (h *BillHandler) Void(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	var req voidReq
	_ = c.ShouldBindJSON(&req)
	vb, err := h.sale.Void(c.Request.Context(), middleware.StoreID(c), id, req.Reason, middleware.UserID(c))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"void_bill": vb})
}

// Receipt returns the receipt as printable HTML (default) or raw ESC/POS bytes.
//
//	GET /bills/:id/receipt?format=html|escpos&width=58|80
func (h *BillHandler) Receipt(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	d, err := h.sale.GetBillDetail(c.Request.Context(), middleware.StoreID(c), id)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	width := 58
	if c.Query("width") == "80" {
		width = 80
	}
	view := toReceiptView(d)
	if c.Query("format") == "escpos" {
		c.Header("Content-Disposition", "attachment; filename=\""+d.Bill.BillNo+".escpos\"")
		c.Data(http.StatusOK, "application/octet-stream", receipt.ESCPOS(view, width))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(receipt.HTML(view, width)))
}

func toReceiptView(d service.BillDetail) receipt.View {
	items := make([]receipt.Line, 0, len(d.Items))
	for _, it := range d.Items {
		items = append(items, receipt.Line{
			Name: it.NameSnapshot, Qty: it.Qty, Price: it.PriceSnapshot, LineTotal: it.LineTotal,
		})
	}
	return receipt.View{
		ShopName:      d.ShopName,
		BillNo:        d.Bill.BillNo,
		Time:          fmtBillTime(d.Bill.CreatedAt),
		Cashier:       d.CashierName,
		Items:         items,
		Subtotal:      d.Bill.Subtotal,
		Discount:      d.Bill.Discount,
		Total:         d.Bill.Total,
		Paid:          d.Bill.Paid,
		Change:        d.Bill.Change,
		PaymentMethod: d.Bill.PaymentMethod,
		Voided:        d.Voided || d.Bill.Status == "void",
	}
}

func billResponse(d service.BillDetail) gin.H {
	return gin.H{
		"bill":      d.Bill,
		"items":     d.Items,
		"shop_name": d.ShopName,
		"cashier":   d.CashierName,
		"voided":    d.Voided,
	}
}

func fmtBillTime(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Local().Format("02/01/2006 15:04")
}
