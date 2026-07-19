package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipEntry struct {
	limiter *rate.Limiter
	seen    time.Time
}

// RateLimit throttles requests per client IP (token bucket). Used to blunt
// brute-force on auth and abusive bursts on the sync endpoint.
func RateLimit(perMinute int, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*ipEntry)
	limit := rate.Limit(float64(perMinute) / 60.0)

	// Evict idle IPs periodically so the map doesn't grow unbounded.
	go func() {
		for range time.Tick(3 * time.Minute) {
			mu.Lock()
			for ip, e := range clients {
				if time.Since(e.seen) > 5*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		e, ok := clients[ip]
		if !ok {
			e = &ipEntry{limiter: rate.NewLimiter(limit, burst)}
			clients[ip] = e
		}
		e.seen = time.Now()
		allowed := e.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "คำขอถี่เกินไป กรุณาลองใหม่อีกครั้ง"})
			return
		}
		c.Next()
	}
}
