package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sabai-pos/backend/internal/auth"
	"sabai-pos/backend/internal/domain"
)

const (
	ctxUserID  = "user_id"
	ctxStoreID = "store_id"
	ctxRole    = "role"
)

// Auth validates the Bearer access token and puts user id, store id and role
// into the request context.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := auth.ParseAccessToken(secret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		uid, err1 := uuid.Parse(claims.Subject)
		sid, err2 := uuid.Parse(claims.StoreID)
		if err1 != nil || err2 != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "malformed token claims"})
			return
		}
		c.Set(ctxUserID, uid)
		c.Set(ctxStoreID, sid)
		c.Set(ctxRole, domain.Role(claims.Role))
		c.Next()
	}
}

// RequireRole allows the request only if the caller's role is in roles.
// Authorisation is enforced here on the server, never by hiding UI.
func RequireRole(roles ...domain.Role) gin.HandlerFunc {
	allowed := make(map[domain.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		if _, ok := allowed[UserRole(c)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
			return
		}
		c.Next()
	}
}

// NotifySecret guards the payment-notification webhook with a shared secret
// (the phone forwarder isn't a JWT client). Empty secret => feature disabled.
func NotifySecret(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "payment notify not configured"})
			return
		}
		if c.GetHeader("X-Notify-Secret") != secret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bad secret"})
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) uuid.UUID  { v, _ := c.Get(ctxUserID); id, _ := v.(uuid.UUID); return id }
func StoreID(c *gin.Context) uuid.UUID { v, _ := c.Get(ctxStoreID); id, _ := v.(uuid.UUID); return id }
func UserRole(c *gin.Context) domain.Role {
	v, _ := c.Get(ctxRole)
	r, _ := v.(domain.Role)
	return r
}
