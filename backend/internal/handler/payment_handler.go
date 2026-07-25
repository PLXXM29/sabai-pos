package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sabai-pos/backend/internal/config"
	"sabai-pos/backend/internal/domain"
	"sabai-pos/backend/internal/middleware"
	"sabai-pos/backend/internal/service"
)

type PaymentHandler struct {
	svc     *service.PaymentService
	cfg     *config.Config
	log     *zap.Logger
	amountR *regexp.Regexp // optional custom amount parser (one capture group)
}

func NewPaymentHandler(svc *service.PaymentService, cfg *config.Config, log *zap.Logger) *PaymentHandler {
	var r *regexp.Regexp
	if cfg.LineAmountRegex != "" {
		r, _ = regexp.Compile(cfg.LineAmountRegex)
	}
	return &PaymentHandler{svc: svc, cfg: cfg, log: log, amountR: r}
}

type createPaymentReq struct {
	Amount     int64   `json:"amount"` // satang
	ClientUUID *string `json:"bill_client_uuid"`
}

func (h *PaymentHandler) Create(c *gin.Context) {
	var req createPaymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	var billUUID *uuid.UUID
	if req.ClientUUID != nil && *req.ClientUUID != "" {
		if u, err := uuid.Parse(*req.ClientUUID); err == nil {
			billUUID = &u
		}
	}
	p, err := h.svc.Create(c.Request.Context(), middleware.StoreID(c), req.Amount, billUUID)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": p.ID, "amount": p.Amount, "status": p.Status})
}

func (h *PaymentHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	p, err := h.svc.Get(c.Request.Context(), middleware.StoreID(c), id)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": p.ID, "status": p.Status, "amount": p.Amount})
}

func (h *PaymentHandler) Cancel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		writeError(c, h.log, domain.Validation("id ไม่ถูกต้อง"))
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), middleware.StoreID(c), id); err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// notifyReq accepts the amount as satang, as baht, OR as raw notification text
// (the server extracts the amount) — so a phone forwarder can just POST the
// whole SCB Connect message.
type notifyReq struct {
	Amount       float64 `json:"amount"`        // baht
	AmountSatang *int64  `json:"amount_satang"` // optional, exact
	Text         string  `json:"text"`          // raw notification; amount parsed out
	Ref          string  `json:"ref"`
	Note         string  `json:"note"`
}

// Notify is called by the phone/forwarder (guarded by X-Notify-Secret).
func (h *PaymentHandler) Notify(c *gin.Context) {
	var req notifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	var satang int64
	switch {
	case req.AmountSatang != nil:
		satang = *req.AmountSatang
	case req.Amount > 0:
		satang = int64(math.Round(req.Amount * 100))
	case req.Text != "":
		if s, ok := parseBahtToSatang(req.Text, h.amountR); ok {
			satang = s
		}
	}
	if satang <= 0 {
		writeError(c, h.log, domain.Validation("อ่านยอดเงินไม่ได้"))
		return
	}
	note := req.Note
	if note == "" {
		note = req.Text
	}
	matched, err := h.svc.Notify(c.Request.Context(), satang, req.Ref, note)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"matched": matched})
}

// LineWebhook receives events from a LINE Official Account (Messaging API).
// It verifies the signature, then extracts a baht amount from each text message
// and matches it to a pending payment. Works on iOS + Android (unlike reading
// bank notifications on a phone).
func (h *PaymentHandler) LineWebhook(c *gin.Context) {
	if h.cfg.LineChannelSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LINE not configured"})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad body"})
		return
	}
	if !verifyLineSignature(h.cfg.LineChannelSecret, body, c.GetHeader("x-line-signature")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bad signature"})
		return
	}
	var payload struct {
		Events []struct {
			Type    string `json:"type"`
			Message struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"events"`
	}
	_ = json.Unmarshal(body, &payload)

	for _, e := range payload.Events {
		if e.Type != "message" || e.Message.Type != "text" {
			continue
		}
		if satang, ok := parseBahtToSatang(e.Message.Text, h.amountR); ok {
			if _, err := h.svc.Notify(c.Request.Context(), satang, "line", e.Message.Text); err != nil {
				h.log.Error("line notify failed", zap.Error(err))
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true}) // LINE expects 200
}

func verifyLineSignature(secret string, body []byte, header string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(header))
}

// default money-in patterns: a number (optional thousands, 2 decimals) that
// follows a "received/credit" keyword; falls back to the first such number.
var (
	lineAmountRe  = regexp.MustCompile(`[0-9][0-9,]*\.[0-9]{2}`)
	lineKeywordRe = regexp.MustCompile(`(?i)(รับ|เข้า|โอน|เครดิต|credit|deposit|received|\+)\s*[^0-9]{0,6}([0-9][0-9,]*\.[0-9]{2})`)
)

// parseBahtToSatang extracts a baht amount from a message and returns satang.
func parseBahtToSatang(text string, custom *regexp.Regexp) (int64, bool) {
	pick := ""
	if custom != nil {
		if m := custom.FindStringSubmatch(text); len(m) >= 2 {
			pick = m[1]
		}
	}
	if pick == "" {
		if m := lineKeywordRe.FindStringSubmatch(text); len(m) >= 3 {
			pick = m[2]
		}
	}
	if pick == "" {
		pick = lineAmountRe.FindString(text)
	}
	if pick == "" {
		return 0, false
	}
	baht := strings.ReplaceAll(pick, ",", "")
	// baht like "1234.56" -> satang int, no float rounding drift
	dot := strings.IndexByte(baht, '.')
	whole := baht[:dot]
	frac := baht[dot+1:]
	var satang int64
	for _, ch := range whole {
		satang = satang*10 + int64(ch-'0')
	}
	satang *= 100
	satang += int64(frac[0]-'0')*10 + int64(frac[1]-'0')
	return satang, satang > 0
}
