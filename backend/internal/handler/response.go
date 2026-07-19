package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minimart-pos/backend/internal/domain"
)

// writeError maps a domain error kind to an HTTP status. Unknown errors are
// 500 and logged (their detail is never leaked to the client).
func writeError(c *gin.Context, log *zap.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		reqID, _ := c.Get("request_id")
		log.Error("unhandled_error", zap.Error(err), zap.Any("request_id", reqID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
