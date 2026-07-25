// Package api wires config, middleware, services and handlers into a router.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sabai-pos/backend/internal/config"
	"sabai-pos/backend/internal/demo"
	"sabai-pos/backend/internal/domain"
	"sabai-pos/backend/internal/handler"
	"sabai-pos/backend/internal/middleware"
	"sabai-pos/backend/internal/service"
	"sabai-pos/backend/internal/store"
	"sabai-pos/backend/internal/web"
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
	payments := service.NewPaymentService(st)

	// Handlers.
	authH := handler.NewAuthHandler(authSvc, cfg, log)
	prodH := handler.NewProductHandler(catalog, stock, log)
	billH := handler.NewBillHandler(sale, log)
	reportH := handler.NewReportHandler(reports, log)
	payH := handler.NewPaymentHandler(payments, cfg, log)
	metaH := handler.NewMetaHandler(cfg, demo.New(st.Pool()), log)

	v1 := r.Group("/api/v1")

	// Deployment identity — what the UI reads before drawing the sign-in screen.
	v1.GET("/meta", metaH.Meta)

	// The showcase dataset is rebuildable by anyone who can reach it, because
	// visitors are meant to change things. The route only exists in demo mode:
	// a real store cannot have its books truncated by an unauthenticated POST.
	if cfg.DemoMode {
		v1.POST("/demo/reset", middleware.RateLimit(6, 2), metaH.ResetDemo)
	}

	// Public auth endpoints — rate limited against brute force (per IP).
	authLimit := middleware.RateLimit(20, 10)
	v1.POST("/auth/login", authLimit, authH.Login)
	v1.POST("/auth/refresh", authLimit, authH.Refresh)
	v1.POST("/auth/logout", authH.Logout)

	// Payment-received webhook — called by the phone/LINE forwarder, guarded by a
	// shared secret (not a JWT client). Rate limited. (Separate path so it doesn't
	// clash with the /payments/:id wildcard.)
	v1.POST("/webhooks/payment",
		middleware.RateLimit(120, 40),
		middleware.NotifySecret(cfg.PaymentNotifySecret),
		payH.Notify)

	// LINE Official Account webhook (signature-verified inside the handler).
	v1.POST("/webhooks/line", middleware.RateLimit(120, 40), payH.LineWebhook)

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

		// Payment intents (auto-confirm transfer): cashier opens + polls.
		authed.POST("/payments", payH.Create)
		authed.GET("/payments/:id", payH.Get)
		authed.POST("/payments/:id/cancel", payH.Cancel)

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

	// The compiled UI ships inside this binary, so whatever the API did not
	// match is either a static asset or a client-side route (deep links and hard
	// refreshes must both land on the app shell). Registered last, as the
	// fallback, so it can never shadow a real endpoint — and requests still
	// under /api get a JSON 404 instead of an HTML page.
	if cfg.ServeUI {
		r.NoRoute(gin.WrapF(web.Handler(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		})))
	}

	return r
}
