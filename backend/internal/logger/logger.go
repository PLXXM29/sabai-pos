// Package logger builds the application's structured logger (Zap).
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a production or development logger depending on appEnv, at the
// given level. Development uses human-friendly console output; production uses
// JSON for log aggregation.
func New(appEnv, level string) (*zap.Logger, error) {
	lvl := zap.NewAtomicLevelAt(parseLevel(level))

	var cfg zap.Config
	if appEnv == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	cfg.Level = lvl
	return cfg.Build()
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
