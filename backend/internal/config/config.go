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
	AppEnv string // development | production
	Port   string // HTTP listen port

	DatabaseURL     string
	DBMaxConns      int32
	DBConnTimeout   time.Duration
	ShutdownTimeout time.Duration

	LogLevel string // debug | info | warn | error

	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration

	CORSAllowedOrigins []string
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// Load reads config from the environment and validates it. Any problem is
// returned as an error so main() can exit before serving a single request.
func Load() (*Config, error) {
	c := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		Port:               getEnv("HTTP_PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		DBMaxConns:         int32(getEnvInt("DB_MAX_CONNS", 10)),
		DBConnTimeout:      getEnvDuration("DB_CONN_TIMEOUT", 5*time.Second),
		ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		JWTAccessSecret:    os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:   os.Getenv("JWT_REFRESH_SECRET"),
		AccessTokenTTL:     getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:    getEnvDuration("REFRESH_TOKEN_TTL", 720*time.Hour), // 30d
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
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
