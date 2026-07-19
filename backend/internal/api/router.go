// Package api wires config, middleware, services and handlers into a router.
package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minimart-pos/backend/internal/config"
	"minimart-pos/backend/internal/domain"
	"minimart-pos/backend/internal/handler"
	"minimart-pos/backend/internal/middleware"
	"minimart-pos/backend/internal/service"
	"minimart-pos/backend/internal/store"
)

func NewRouter(cfg *config.Config, log *zap.Logger, st *store.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.SecureHeaders())
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// Health / readiness (unauthenticated).
	health := handler.NewHealthHandler(st.Pool())
	r.GET("/healthz", health.Live)
	r.GET("/readyz", health.Ready)

	// Services.
	authSvc := service.NewAuthService(st, cfg)
	catalog := service.NewCatalogService(st)
	stock := service.NewStockService(st)
	sale := service.NewSaleService(st)
	reports := service.NewReportService(st)

	// Handlers.
	authH := handler.NewAuthHandler(authSvc, cfg, log)
	prodH := handler.NewProductHandler(catalog, stock, log)
	billH := handler.NewBillHandler(sale, log)
	reportH := handler.NewReportHandler(reports, log)

	v1 := r.Group("/api/v1")

	// Public auth endpoints — rate limited against brute force (per IP).
	authLimit := middleware.RateLimit(20, 10)
	v1.POST("/auth/login", authLimit, authH.Login)
	v1.POST("/auth/refresh", authLimit, authH.Refresh)
	v1.POST("/auth/logout", authH.Logout)

	// Authenticated endpoints.
	authed := v1.Group("")
	authed.Use(middleware.Auth(cfg.JWTAccessSecret))
	{
		authed.GET("/auth/me", authH.Me)
		authed.POST("/auth/change-password", authH.ChangePassword)
		authed.POST("/auth/register",
			middleware.RequireRole(domain.RoleSuperadmin, domain.RoleManager), authH.Register)

		// Reads: any authenticated role (incl. cashier).
		authed.GET("/products", prodH.List)
		authed.GET("/products/:id", prodH.Get)
		authed.GET("/products/:id/onhand", prodH.OnHand)

		// Sales: cashiers sell + view/print receipts. The checkout (sync target)
		// is rate limited generously — well above a busy till's real rate.
		authed.POST("/bills", middleware.RateLimit(120, 40), billH.Checkout)
		authed.GET("/bills/:id", billH.Get)
		authed.GET("/bills/:id/receipt", billH.Receipt)

		// Writes / sensitive ops: manager+ only (RBAC enforced server-side).
		manage := authed.Group("")
		manage.Use(middleware.RequireRole(domain.RoleSuperadmin, domain.RoleManager))
		{
			manage.POST("/products", prodH.Create)
			manage.PUT("/products/:id", prodH.Update)
			manage.DELETE("/products/:id", prodH.Delete)
			manage.POST("/products/:id/receive", prodH.Receive)
			manage.POST("/bills/:id/void", billH.Void)

			// Reports (profit/margin) — manager+ only.
			manage.GET("/reports/summary", reportH.Summary)
			manage.GET("/reports/top-products", reportH.TopProducts)
			manage.GET("/reports/sales-daily", reportH.SalesDaily)
		}
	}

	return r
}
