// Command api is the Sabai POS server: HTTP API, schema migrations and the
// compiled web UI in a single binary, so a deployment is one container.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata" // embed the timezone database (distroless has none)

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"sabai-pos/backend/internal/api"
	"sabai-pos/backend/internal/config"
	"sabai-pos/backend/internal/dbmigrate"
	"sabai-pos/backend/internal/demo"
	"sabai-pos/backend/internal/logger"
	"sabai-pos/backend/internal/store"
	"sabai-pos/backend/internal/web"
)

func main() {
	// 1. Config — fail fast on anything invalid.
	cfg, err := config.Load()
	if err != nil {
		// Logger not up yet; use stdlib to guarantee the message is seen.
		os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	// 2. Logger.
	log, err := logger.New(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		os.Stderr.WriteString("logger error: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	// 3. Database pool.
	rootCtx := context.Background()
	pool, err := connectDB(rootCtx, cfg, log)
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()
	log.Info("database connected", zap.Int32("max_conns", cfg.DBMaxConns))

	// 4. Schema. Migrated in-process because the deployment target has no
	// separate migrate step; golang-migrate's advisory lock keeps that safe when
	// more than one replica boots at once.
	if cfg.AutoMigrate {
		version, changed, err := dbmigrate.Up(cfg.DatabaseURL)
		if err != nil {
			log.Fatal("migration failed", zap.Error(err))
		}
		log.Info("schema ready", zap.Uint("version", version), zap.Bool("migrated", changed))
	}

	db := store.NewDB(pool)

	// 5. Demo dataset. This path only ever populates an empty database; wiping a
	// populated one is an explicit POST to /api/v1/demo/reset or the timer below.
	if cfg.DemoMode {
		seeder := demo.New(pool)
		res, seeded, err := seeder.Ensure(rootCtx, demo.DefaultHistoryDays)
		if err != nil {
			log.Fatal("demo seed failed", zap.Error(err))
		}
		if seeded {
			log.Info("demo dataset created",
				zap.Int("products", res.Products), zap.Int("bills", res.Bills),
				zap.Int("history_days", res.HistoryDays), zap.String("took", res.Took))
		}
		if cfg.DemoResetEvery > 0 {
			// Checked here as well as on the timer: between visitors this
			// process does not exist, so boot is often the only chance a
			// scheduled rebuild gets to run at all.
			refreshIfStale(rootCtx, seeder, cfg.DemoResetEvery, log)
			go resetPeriodically(rootCtx, seeder, cfg.DemoResetEvery, log)
		}
	}

	// 6. Router.
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := api.NewRouter(cfg, log, db)
	if cfg.ServeUI && !web.Bundled() {
		log.Warn("no UI bundled in this binary — serving the API only " +
			"(build the frontend first, or set SERVE_UI=false)")
	}

	// 7. HTTP server with graceful shutdown.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("http server listening",
			zap.String("port", cfg.Port), zap.String("env", cfg.AppEnv),
			zap.Bool("ui", cfg.ServeUI && web.Bundled()), zap.Bool("demo", cfg.DemoMode))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("http server error", zap.Error(err))
		}
	}()

	// Wait for SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown signal received, draining connections")

	ctx, cancel := context.WithTimeout(rootCtx, cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("server stopped cleanly")
}

// connectDB dials the database, retrying briefly. The retry is not defensive
// padding: the demo runs on serverless Postgres that suspends when idle, and a
// cold start can outlast a single connect timeout. Failing the boot on that
// would turn a two-second wake-up into a crash loop.
func connectDB(ctx context.Context, cfg *config.Config, log *zap.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.DBMaxConns

	const attempts = 5
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		pool, err := openPool(ctx, poolCfg, cfg.DBConnTimeout)
		if err == nil {
			return pool, nil
		}
		lastErr = err
		if attempt == attempts {
			break
		}
		wait := time.Duration(attempt) * time.Second
		log.Warn("database not ready, retrying",
			zap.Int("attempt", attempt), zap.Duration("in", wait), zap.Error(err))
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

func openPool(ctx context.Context, poolCfg *pgxpool.Config, timeout time.Duration) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// resetPeriodically returns the shared demo to its designed state, so yesterday's
// visitors do not decide what today's visitors see.
//
// The tick is only a prompt to re-read the clock — staleness is measured against
// a timestamp in the database, because this process is stopped whenever nobody
// is looking and cannot be trusted to have been running for a day.
func resetPeriodically(ctx context.Context, seeder *demo.Seeder, every time.Duration, log *zap.Logger) {
	interval := every
	if interval > time.Hour {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshIfStale(ctx, seeder, every, log)
		}
	}
}

func refreshIfStale(ctx context.Context, seeder *demo.Seeder, maxAge time.Duration, log *zap.Logger) {
	res, rebuilt, err := seeder.RefreshIfStale(ctx, maxAge, demo.DefaultHistoryDays)
	if err != nil {
		log.Error("scheduled demo reset failed", zap.Error(err))
		return
	}
	if rebuilt {
		log.Info("demo dataset rebuilt on schedule",
			zap.Duration("max_age", maxAge), zap.Int("bills", res.Bills),
			zap.String("took", res.Took))
	}
}
