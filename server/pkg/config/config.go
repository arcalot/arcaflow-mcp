// Package config provides configuration loading and validation.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for the MCP server.
type Config struct {
	Mode         string          `yaml:"mode"`
	Address      string          `yaml:"address"`
	Logging      LoggingConfig   `yaml:"logging"`
	Auth         AuthConfig      `yaml:"auth"`
	RateLimiting RateLimitConfig `yaml:"rate_limit"`
	Tenancy      TenancyConfig   `yaml:"tenancy"`
}

// LoggingConfig controls structured logging behavior.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// AuthConfig controls authentication for server mode.
type AuthConfig struct {
	AdminToken string `yaml:"admin_token"`
}

// RateLimitConfig controls server mode rate limiting.
type RateLimitConfig struct {
	Enabled            bool `yaml:"enabled"`
	RequestsPerMinute  int  `yaml:"requests_per_minute"`
	WindowSeconds      int  `yaml:"window_seconds"`
	BackoffEnabled     bool `yaml:"backoff_enabled"`
	BackoffBaseSeconds int  `yaml:"backoff_base_seconds"`
	BackoffMaxSeconds  int  `yaml:"backoff_max_seconds"`
}

// TenancyConfig controls per-tenant isolation settings.
type TenancyConfig struct {
	WorkspaceRoot         string `yaml:"workspace_root"`
	MaxConcurrentRequests int    `yaml:"max_concurrent_requests"`
	MaxSessions           int    `yaml:"max_sessions"`
}

// Default returns a baseline configuration.
func Default() Config {
	workspaceRoot := filepath.Join(os.TempDir(), "arcaflow-mcp", "tenants")
	return Config{
		Mode:    "local",
		Address: "127.0.0.1:8080",
		Logging: LoggingConfig{Level: "info"},
		Auth:    AuthConfig{},
		RateLimiting: RateLimitConfig{
			Enabled:            true,
			RequestsPerMinute:  60,
			WindowSeconds:      60,
			BackoffEnabled:     true,
			BackoffBaseSeconds: 1,
			BackoffMaxSeconds:  60,
		},
		Tenancy: TenancyConfig{
			WorkspaceRoot:         workspaceRoot,
			MaxConcurrentRequests: 10,
			MaxSessions:           4,
		},
	}
}

// Load reads configuration from a YAML file, then applies environment overrides.
func Load(path string) (Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config: %w", err)
		}
	}

	applyEnvOverrides(&cfg)

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_MODE"); ok && value != "" {
		cfg.Mode = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_ADDRESS"); ok && value != "" {
		cfg.Address = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_LOG_LEVEL"); ok && value != "" {
		cfg.Logging.Level = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_ADMIN_TOKEN"); ok && value != "" {
		cfg.Auth.AdminToken = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_RATE_LIMIT_ENABLED"); ok {
		cfg.RateLimiting.Enabled = value == "1" || value == "true"
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_RATE_LIMIT_RPM"); ok && value != "" {
		cfg.RateLimiting.RequestsPerMinute, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_RATE_LIMIT_WINDOW_SECONDS"); ok && value != "" {
		cfg.RateLimiting.WindowSeconds, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_ENABLED"); ok {
		cfg.RateLimiting.BackoffEnabled = value == "1" || value == "true"
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_BASE_SECONDS"); ok && value != "" {
		cfg.RateLimiting.BackoffBaseSeconds, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_MAX_SECONDS"); ok && value != "" {
		cfg.RateLimiting.BackoffMaxSeconds, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT"); ok && value != "" {
		cfg.Tenancy.WorkspaceRoot = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_MAX_CONCURRENT"); ok && value != "" {
		cfg.Tenancy.MaxConcurrentRequests, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_MAX_SESSIONS"); ok && value != "" {
		cfg.Tenancy.MaxSessions, _ = strconv.Atoi(value)
	}
}

// Validate checks required fields and constraints.
func Validate(cfg Config) error {
	switch cfg.Mode {
	case "local", "server":
	default:
		return errors.New("mode must be local or server")
	}

	if cfg.Address == "" {
		return errors.New("address must not be empty")
	}

	if cfg.Mode == "server" && cfg.Auth.AdminToken == "" {
		return errors.New("admin token must be set in server mode")
	}
	if cfg.Mode == "server" && cfg.RateLimiting.Enabled {
		if cfg.RateLimiting.RequestsPerMinute <= 0 {
			return errors.New("rate limit requests_per_minute must be greater than zero")
		}
		if cfg.RateLimiting.WindowSeconds <= 0 {
			return errors.New("rate limit window_seconds must be greater than zero")
		}
		if cfg.RateLimiting.BackoffEnabled {
			if cfg.RateLimiting.BackoffBaseSeconds <= 0 {
				return errors.New(
					"rate limit backoff_base_seconds must be greater than zero",
				)
			}
			if cfg.RateLimiting.BackoffMaxSeconds <= 0 {
				return errors.New(
					"rate limit backoff_max_seconds must be greater than zero",
				)
			}
			if cfg.RateLimiting.BackoffMaxSeconds <
				cfg.RateLimiting.BackoffBaseSeconds {
				return errors.New(
					"rate limit backoff_max_seconds must be >= backoff_base_seconds",
				)
			}
		}
	}
	if cfg.Mode == "server" {
		if cfg.Tenancy.WorkspaceRoot == "" {
			return errors.New("tenancy workspace_root must be set in server mode")
		}
		if cfg.Tenancy.MaxConcurrentRequests < 0 {
			return errors.New("tenancy max_concurrent_requests must be >= 0")
		}
		if cfg.Tenancy.MaxSessions < 0 {
			return errors.New("tenancy max_sessions must be >= 0")
		}
	}

	return nil
}
