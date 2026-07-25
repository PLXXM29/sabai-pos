package handler

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sabai-pos/backend/internal/config"
	"sabai-pos/backend/internal/demo"
)

// resetCooldown throttles rebuilds globally, on top of the per-IP rate limit.
// Rebuilding truncates and reloads every business table; letting a handful of
// clients interleave those would turn a demo into a self-inflicted load test.
const resetCooldown = 45 * time.Second

// MetaHandler answers the one thing the static UI bundle cannot know about
// itself: which deployment is it talking to. The bundle is built once and run
// anywhere, so "am I the public demo?" has to be a runtime answer, not a
// compile-time flag baked into the JavaScript.
type MetaHandler struct {
	cfg    *config.Config
	seeder *demo.Seeder
	log    *zap.Logger

	mu        sync.Mutex
	lastReset time.Time
}

func NewMetaHandler(cfg *config.Config, seeder *demo.Seeder, log *zap.Logger) *MetaHandler {
	return &MetaHandler{cfg: cfg, seeder: seeder, log: log}
}

type metaResponse struct {
	Version string `json:"version"`
	Demo    bool   `json:"demo"`
	// Accounts is populated in demo mode only. Publishing sign-in details is
	// obviously wrong for a real store and exactly right for a showcase: the
	// alternative is printing them in the HTML, where they would ship to every
	// deployment built from the same bundle.
	Accounts   []demo.Account `json:"accounts,omitempty"`
	ResetEvery string         `json:"reset_every,omitempty"`
}

func (h *MetaHandler) Meta(c *gin.Context) {
	res := metaResponse{Version: h.cfg.Version, Demo: h.cfg.DemoMode}
	if h.cfg.DemoMode {
		res.Accounts = demo.Accounts
		if h.cfg.DemoResetEvery > 0 {
			res.ResetEvery = h.cfg.DemoResetEvery.String()
		}
	}
	// Deployment identity changes only on deploy, but a stale cached copy would
	// strand the sign-in screen on the wrong mode after one.
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

// ResetDemo rebuilds the sample dataset. Unauthenticated by design — a visitor
// who has just deleted half the catalogue needs to be able to put it back — but
// mounted only when DEMO_MODE is on, so a real deployment has no such route.
func (h *MetaHandler) ResetDemo(c *gin.Context) {
	h.mu.Lock()
	if wait := resetCooldown - time.Since(h.lastReset); wait > 0 {
		h.mu.Unlock()
		c.Header("Retry-After", formatSeconds(wait))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "เพิ่งรีเซ็ตข้อมูลไปเมื่อสักครู่ กรุณารออีก " + formatSeconds(wait) + " วินาที",
		})
		return
	}
	h.lastReset = time.Now()
	h.mu.Unlock()

	res, err := h.seeder.Reset(c.Request.Context(), demo.DefaultHistoryDays)
	if err != nil {
		// Let the next caller retry immediately rather than serving a cooldown
		// for a rebuild that never happened.
		h.mu.Lock()
		h.lastReset = time.Time{}
		h.mu.Unlock()
		writeError(c, h.log, err)
		return
	}
	h.log.Info("demo_dataset_reset",
		zap.Int("bills", res.Bills), zap.Int("products", res.Products), zap.String("took", res.Took))
	c.JSON(http.StatusOK, gin.H{"status": "ok", "dataset": res})
}

// formatSeconds rounds up, so a 0.4s remainder reads as "1" rather than "0".
func formatSeconds(d time.Duration) string {
	s := int((d + time.Second - 1) / time.Second)
	if s < 1 {
		s = 1
	}
	return strconv.Itoa(s)
}
