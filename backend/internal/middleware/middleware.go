// Package middleware holds cross-cutting HTTP concerns: request id, structured
// access logging, panic recovery, and CORS.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const RequestIDHeader = "X-Request-ID"

// SecureHeaders sets baseline hardening headers on every response.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-XSS-Protection", "0")
		c.Next()
	}
}

// RequestID attaches a request id (from the client header or freshly generated)
// to the context and response so logs and clients can correlate a call.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = newID()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

// Logger logs one structured line per request with method, path, status,
// latency and request id.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		reqID, _ := c.Get("request_id")
		log.Info("http_request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.Any("request_id", reqID),
		)
	}
}

// Recovery converts a panic into a 500 and logs it with the request id instead
// of crashing the server.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID, _ := c.Get("request_id")
				log.Error("panic_recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.Any("request_id", reqID),
				)
				c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
			}
		}()
		c.Next()
	}
}

// CORS allows only the configured origins.
func CORS(allowed []string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allow[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allow[origin]; ok {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,"+RequestIDHeader)
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "req-" + time.Now().Format("20060102150405.000000")
	}
	return hex.EncodeToString(b)
}
