package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sabai-pos/backend/internal/domain"
	"sabai-pos/backend/internal/middleware"
	"sabai-pos/backend/internal/service"
)

type ProductHandler struct {
	catalog *service.CatalogService
	stock   *service.StockService
	log     *zap.Logger
}

func NewProductHandler(catalog *service.CatalogService, stock *service.StockService, log *zap.Logger) *ProductHandler {
	return &ProductHandler{catalog: catalog, stock: stock, log: log}
}

// productReq — money fields are satang (integer). See ADR 0001.
type productReq struct {
	Name         string  `json:"name"`
	Barcode      *string `json:"barcode"`
	Category     string  `json:"category"`
	CostPrice    int64   `json:"cost_price"`
	SellPrice    int64   `json:"sell_price"`
	ReorderPoint int32   `json:"reorder_point"`
}

func (r productReq) toInput() service.ProductInput {
	return service.ProductInput{
		Name:         r.Name,
		Barcode:      r.Barcode,
		Category:     r.Category,
		CostPrice:    r.CostPrice,
		SellPrice:    r.SellPrice,
		ReorderPoint: r.ReorderPoint,
	}
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func (h *ProductHandler) List(c *gin.Context) {
	items, err := h.catalog.List(c.Request.Context(), middleware.StoreID(c))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": items})
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	p, err := h.catalog.Get(c.Request.Context(), middleware.StoreID(c), id)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": p})
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	p, err := h.catalog.Create(c.Request.Context(), middleware.StoreID(c), req.toInput())
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"product": p})
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	p, err := h.catalog.Update(c.Request.Context(), middleware.StoreID(c), id, req.toInput())
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": p})
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	if err := h.catalog.Deactivate(c.Request.Context(), middleware.StoreID(c), id); err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type receiveReq struct {
	Qty        int32   `json:"qty"`
	Reason     string  `json:"reason"`
	ClientUUID *string `json:"client_uuid"`
}

// Receive stock into a product (append-only ledger + inventory, atomic).
func (h *ProductHandler) Receive(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	var req receiveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
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
	inv, err := h.stock.Receive(c.Request.Context(), middleware.StoreID(c), id, req.Qty, req.Reason, middleware.UserID(c), clientUUID)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"inventory": inv})
}

func (h *ProductHandler) OnHand(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	n, err := h.stock.OnHand(c.Request.Context(), id)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_id": id, "on_hand": n})
}
