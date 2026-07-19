// Command api is the MiniMart POS HTTP server entrypoint.
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

	"minimart-pos/backend/internal/api"
	"minimart-pos/backend/internal/config"
	"minimart-pos/backend/internal/logger"
	"minimart-pos/backend/internal/store"
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
	pool, err := connectDB(rootCtx, cfg)
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()
	log.Info("database connected", zap.Int32("max_conns", cfg.DBMaxConns))

	// 4. Router.
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := api.NewRouter(cfg, log, store.NewDB(pool))

	// 5. HTTP server with graceful shutdown.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("http server listening", zap.String("port", cfg.Port), zap.String("env", cfg.AppEnv))
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

func connectDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.DBMaxConns

	ctx, cancel := context.WithTimeout(ctx, cfg.DBConnTimeout)
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
