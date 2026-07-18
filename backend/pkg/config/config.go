// Package config provides application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds the application configuration.
type Config struct {
	// Port is the HTTP server listen port.
	Port int
	// LogLevel is the zap log level.
	LogLevel zapcore.Level
	// Environment is the deployment environment.
	Environment string
	// RulesDir is the path to rule YAML files (base layer).
	RulesDir string
	// RulesProjectDir is optional project-specific rules layer.
	RulesProjectDir string
	// RulesOverrideDir is optional per-env rules override layer.
	RulesOverrideDir string
	// DBDriver is "sqlite" or "postgres". Default: "sqlite".
	DBDriver string
	// DBDSN is the database connection string.
	DBDSN string
	// PluginsDir is the directory with plugin binaries.
	PluginsDir string
	// ParagraphSeparator separates paragraphs in input text. Default: "\n\n".
	ParagraphSeparator string
	// StaticDir is the directory with static frontend files.
	// Default: "../frontend/out" for development.
	StaticDir string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	port, err := getEnvInt("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logLevelStr := os.Getenv("LOG_LEVEL")
	var logLevel zapcore.Level
	if err := logLevel.Set(logLevelStr); err != nil {
		logLevel = zapcore.InfoLevel
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	rulesDir := os.Getenv("RULES_DIR")
	if rulesDir == "" {
		rulesDir = "./rules"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../frontend/out"
	}

	return &Config{
		Port:               port,
		LogLevel:           logLevel,
		Environment:        env,
		RulesDir:           rulesDir,
		RulesProjectDir:    os.Getenv("RULES_PROJECT_DIR"),
		RulesOverrideDir:   os.Getenv("RULES_OVERRIDE_DIR"),
		PluginsDir:         os.Getenv("PLUGINS_DIR"),
		StaticDir:          staticDir,
		DBDriver:           getEnvDefault("DB_DRIVER", "sqlite"),
		DBDSN:              getEnvDefault("DB_DSN", "file:redpolitika.db?cache=shared&_journal_mode=WAL"),
		ParagraphSeparator: os.Getenv("PARAGRAPH_SEPARATOR"),
	}, nil
}

func getEnvDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

// ProvideConfig is an FX provider for Config.
func ProvideConfig() (*Config, error) {
	return Load()
}

// ProvideLogger creates a zap.Logger from Config.
func ProvideLogger(cfg *Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = zap.NewAtomicLevelAt(cfg.LogLevel)
	zapCfg.EncoderConfig.TimeKey = "timestamp"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}
	return logger, nil
}
