package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minimart-pos/backend/internal/middleware"
	"minimart-pos/backend/internal/service"
)

type ReportHandler struct {
	svc *service.ReportService
	log *zap.Logger
}

func NewReportHandler(svc *service.ReportService, log *zap.Logger) *ReportHandler {
	return &ReportHandler{svc: svc, log: log}
}

func (h *ReportHandler) Summary(c *gin.Context) {
	s, err := h.svc.Summary(c.Request.Context(), middleware.StoreID(c))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *ReportHandler) TopProducts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	items, err := h.svc.Top(c.Request.Context(), middleware.StoreID(c), int32(limit))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": items})
}

func (h *ReportHandler) SalesDaily(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	items, err := h.svc.Daily(c.Request.Context(), middleware.StoreID(c), days)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": items})
}
