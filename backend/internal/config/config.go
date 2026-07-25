// Package config loads and validates all runtime configuration from the
// environment. It fails fast: if anything required is missing or malformed,
// Load returns an error and the process must not start.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv  string // development | production
	Port    string // HTTP listen port
	Version string // build identifier, surfaced by /api/v1/meta

	DatabaseURL     string
	DBMaxConns      int32
	DBConnTimeout   time.Duration
	ShutdownTimeout time.Duration

	// AutoMigrate applies pending schema migrations during startup. On by
	// default because the production target is one container with no separate
	// migrate step; turn it off where a DBA owns the schema.
	AutoMigrate bool

	// ServeUI serves the embedded SPA build from this binary. Off means
	// API-only (something else fronts the UI).
	ServeUI bool

	// DemoMode marks this deployment as a public showcase: the login screen
	// advertises its sample accounts and anyone may reset the sample data.
	// It must stay false for a real store.
	DemoMode bool

	// DemoResetEvery re-seeds the demo dataset on a timer so a shared demo
	// heals itself after visitors edit it. Zero disables the timer (the manual
	// reset endpoint still works).
	DemoResetEvery time.Duration

	LogLevel string // debug | info | warn | error

	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration

	CORSAllowedOrigins []string

	// Shared secret the payment-notification forwarder (a phone) must present.
	// Empty = the auto-confirm notify endpoint is disabled.
	PaymentNotifySecret string

	// LINE Official Account Messaging API. LineChannelSecret verifies webhook
	// signatures; LineAmountRegex (optional) overrides the money-in amount parser
	// (must contain one capture group for the baht amount).
	LineChannelSecret string
	LineAmountRegex   string
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// Load reads config from the environment and validates it. Any problem is
// returned as an error so main() can exit before serving a single request.
func Load() (*Config, error) {
	c := &Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnv("HTTP_PORT", "8080"),
		Version:             getEnv("APP_VERSION", "dev"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DBMaxConns:          int32(getEnvInt("DB_MAX_CONNS", 10)),
		DBConnTimeout:       getEnvDuration("DB_CONN_TIMEOUT", 5*time.Second),
		ShutdownTimeout:     getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		AutoMigrate:         getEnvBool("AUTO_MIGRATE", true),
		ServeUI:             getEnvBool("SERVE_UI", true),
		DemoMode:            getEnvBool("DEMO_MODE", false),
		DemoResetEvery:      getEnvDuration("DEMO_RESET_EVERY", 24*time.Hour),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		JWTAccessSecret:     os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:    os.Getenv("JWT_REFRESH_SECRET"),
		AccessTokenTTL:      getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:     getEnvDuration("REFRESH_TOKEN_TTL", 720*time.Hour), // 30d
		CORSAllowedOrigins:  splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		PaymentNotifySecret: os.Getenv("PAYMENT_NOTIFY_SECRET"),
		LineChannelSecret:   os.Getenv("LINE_CHANNEL_SECRET"),
		LineAmountRegex:     os.Getenv("LINE_AMOUNT_REGEX"),
	}

	var problems []string
	if c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if c.JWTAccessSecret == "" {
		problems = append(problems, "JWT_ACCESS_SECRET is required")
	}
	if c.JWTRefreshSecret == "" {
		problems = append(problems, "JWT_REFRESH_SECRET is required")
	}
	if c.IsProduction() {
		if len(c.JWTAccessSecret) < 32 {
			problems = append(problems, "JWT_ACCESS_SECRET must be >= 32 chars in production")
		}
		if len(c.JWTRefreshSecret) < 32 {
			problems = append(problems, "JWT_REFRESH_SECRET must be >= 32 chars in production")
		}
	}
	if c.AppEnv != "development" && c.AppEnv != "production" {
		problems = append(problems, "APP_ENV must be 'development' or 'production'")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, "LOG_LEVEL must be one of debug|info|warn|error")
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return c, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
