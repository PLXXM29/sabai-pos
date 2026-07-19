package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minimart-pos/backend/internal/config"
	"minimart-pos/backend/internal/domain"
	"minimart-pos/backend/internal/middleware"
	"minimart-pos/backend/internal/service"
	"minimart-pos/backend/internal/store"
)

const refreshCookie = "refresh_token"

type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
	log *zap.Logger
}

func NewAuthHandler(svc *service.AuthService, cfg *config.Config, log *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg, log: log}
}

type userView struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	StoreID  string `json:"store_id"`
}

func toUserView(u store.User) userView {
	return userView{ID: u.ID.String(), Username: u.Username, Role: u.Role, StoreID: u.StoreID.String()}
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, raw string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookie, raw, maxAge, "/api/v1/auth", "", h.cfg.IsProduction(), true)
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	h.setRefreshCookie(c, pair.RefreshToken, int(h.cfg.RefreshTokenTTL.Seconds()))
	c.JSON(http.StatusOK, gin.H{
		"access_token": pair.AccessToken,
		"expires_in":   pair.ExpiresIn,
		"token_type":   "Bearer",
		"user":         toUserView(pair.User),
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(refreshCookie)
	if err != nil || raw == "" {
		writeError(c, h.log, domain.ErrUnauthorized)
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), raw)
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	h.setRefreshCookie(c, pair.RefreshToken, int(h.cfg.RefreshTokenTTL.Seconds()))
	c.JSON(http.StatusOK, gin.H{
		"access_token": pair.AccessToken,
		"expires_in":   pair.ExpiresIn,
		"token_type":   "Bearer",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if raw, err := c.Cookie(refreshCookie); err == nil && raw != "" {
		_ = h.svc.Logout(c.Request.Context(), raw)
	}
	h.setRefreshCookie(c, "", -1)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type registerReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Register — manager+ creates a staff account in their own store.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	u, err := h.svc.Register(c.Request.Context(), middleware.StoreID(c), req.Username, req.Password, domain.Role(req.Role))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": toUserView(u)})
}

type changePasswordReq struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword — the logged-in user changes their own password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, h.log, domain.Validation("รูปแบบคำขอไม่ถูกต้อง"))
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), middleware.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	u, err := h.svc.GetUser(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		writeError(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toUserView(u)})
}
