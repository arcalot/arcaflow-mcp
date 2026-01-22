// Package config provides configuration loading and validation.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const defaultDataDir = "/var/lib/arcaflow-mcp"

// Config is the root configuration for the MCP server.
type Config struct {
	Mode         string          `yaml:"mode"`
	Address      string          `yaml:"address"`
	Logging      LoggingConfig   `yaml:"logging"`
	Auth         AuthConfig      `yaml:"auth"`
	RateLimiting RateLimitConfig `yaml:"rate_limit"`
	Tenancy      TenancyConfig   `yaml:"tenancy"`
	Audit        AuditConfig     `yaml:"audit"`
	Usage        UsageConfig     `yaml:"usage"`
	Analysis     AnalysisConfig  `yaml:"analysis"`
}

// LoggingConfig controls structured logging behavior.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// AuthConfig controls authentication for server mode.
type AuthConfig struct {
	AdminToken     string `yaml:"admin_token"`
	TokenStorePath string `yaml:"token_store_path"`
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

// AuditConfig controls audit persistence.
type AuditConfig struct {
	StorePath     string `yaml:"store_path"`
	RetentionDays int    `yaml:"retention_days"`
}

// UsageConfig controls usage statistics persistence.
type UsageConfig struct {
	StorePath string `yaml:"store_path"`
}

// AnalysisConfig controls analysis service integration.
type AnalysisConfig struct {
	HTTPURL string `yaml:"analysis_http_url"`
}

// TenancyConfig controls per-tenant isolation settings.
type TenancyConfig struct {
	WorkspaceRoot         string `yaml:"workspace_root"`
	TenantStorePath       string `yaml:"tenant_store_path"`
	MaxWorkspaceBytes     int64  `yaml:"max_workspace_bytes"`
	MaxRequestCount       int64  `yaml:"max_request_count"`
	MaxConcurrentRequests int    `yaml:"max_concurrent_requests"`
	MaxSessions           int    `yaml:"max_sessions"`
}

// Default returns a baseline configuration.
func Default() Config {
	workspaceRoot := filepath.Join(defaultDataDir, "tenants")
	tokenStorePath := filepath.Join(defaultDataDir, "tokens.json")
	tenantStorePath := filepath.Join(defaultDataDir, "tenants.json")
	auditStorePath := filepath.Join(defaultDataDir, "audit.json")
	usageStorePath := filepath.Join(defaultDataDir, "usage.json")
	return Config{
		Mode:    "local",
		Address: "127.0.0.1:8080",
		Logging: LoggingConfig{Level: "info"},
		Auth: AuthConfig{
			TokenStorePath: tokenStorePath,
		},
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
			TenantStorePath:       tenantStorePath,
			MaxWorkspaceBytes:     0,
			MaxRequestCount:       0,
			MaxConcurrentRequests: 10,
			MaxSessions:           4,
		},
		Audit: AuditConfig{
			StorePath:     auditStorePath,
			RetentionDays: 30,
		},
		Usage: UsageConfig{
			StorePath: usageStorePath,
		},
		Analysis: AnalysisConfig{},
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
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TOKEN_STORE_PATH"); ok && value != "" {
		cfg.Auth.TokenStorePath = value
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
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_STORE_PATH"); ok && value != "" {
		cfg.Tenancy.TenantStorePath = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_MAX_WORKSPACE_BYTES"); ok && value != "" {
		cfg.Tenancy.MaxWorkspaceBytes, _ = strconv.ParseInt(value, 10, 64)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_MAX_REQUESTS"); ok && value != "" {
		cfg.Tenancy.MaxRequestCount, _ = strconv.ParseInt(value, 10, 64)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_MAX_CONCURRENT"); ok && value != "" {
		cfg.Tenancy.MaxConcurrentRequests, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_TENANT_MAX_SESSIONS"); ok && value != "" {
		cfg.Tenancy.MaxSessions, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_AUDIT_STORE_PATH"); ok && value != "" {
		cfg.Audit.StorePath = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_AUDIT_RETENTION_DAYS"); ok && value != "" {
		cfg.Audit.RetentionDays, _ = strconv.Atoi(value)
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_USAGE_STORE_PATH"); ok && value != "" {
		cfg.Usage.StorePath = value
	}
	if value, ok := os.LookupEnv("ARCAFLOW_MCP_ANALYSIS_HTTP_URL"); ok && value != "" {
		cfg.Analysis.HTTPURL = value
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

	if cfg.Analysis.HTTPURL != "" {
		parsed, err := url.Parse(cfg.Analysis.HTTPURL)
		if err != nil {
			return fmt.Errorf("analysis http url invalid: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("analysis http url must use http or https")
		}
		if parsed.Host == "" {
			return errors.New("analysis http url must include host")
		}
	}

	if cfg.Mode == "server" && cfg.Auth.AdminToken == "" {
		return errors.New("admin token must be set in server mode")
	}
	if cfg.Mode == "server" && cfg.Auth.TokenStorePath == "" {
		return errors.New("token store path must be set in server mode")
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
		if cfg.Tenancy.TenantStorePath == "" {
			return errors.New("tenancy tenant_store_path must be set in server mode")
		}
		if cfg.Audit.StorePath == "" {
			return errors.New("audit store_path must be set in server mode")
		}
		if cfg.Usage.StorePath == "" {
			return errors.New("usage store_path must be set in server mode")
		}
		if cfg.Audit.RetentionDays < 0 {
			return errors.New("audit retention_days must be >= 0")
		}
		if cfg.Tenancy.MaxWorkspaceBytes < 0 {
			return errors.New("tenancy max_workspace_bytes must be >= 0")
		}
		if cfg.Tenancy.MaxRequestCount < 0 {
			return errors.New("tenancy max_request_count must be >= 0")
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
