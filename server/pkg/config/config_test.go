package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Mode != "local" {
		t.Fatalf("expected default mode local, got %q", cfg.Mode)
	}
	if cfg.Address == "" {
		t.Fatal("expected default address to be set")
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.Logging.Level)
	}
	if cfg.Auth.AdminToken != "" {
		t.Fatalf("expected default admin token empty, got %q", cfg.Auth.AdminToken)
	}
	if cfg.Auth.TokenStorePath == "" {
		t.Fatalf(
			"expected default token store path set, got %q",
			cfg.Auth.TokenStorePath,
		)
	}
	if cfg.Tenancy.TenantStorePath == "" {
		t.Fatalf("expected default tenant store path set")
	}
	if cfg.Audit.StorePath == "" {
		t.Fatalf("expected default audit store path set")
	}
	if cfg.Usage.StorePath == "" {
		t.Fatalf("expected default usage store path set")
	}
	if cfg.Audit.RetentionDays <= 0 {
		t.Fatalf(
			"expected default audit retention days > 0, got %d",
			cfg.Audit.RetentionDays,
		)
	}
	if cfg.Audit.StorePath == "" {
		t.Fatalf("expected default audit store path set")
	}
	if !cfg.RateLimiting.Enabled {
		t.Fatalf("expected rate limiting enabled by default")
	}
	if cfg.RateLimiting.RequestsPerMinute <= 0 {
		t.Fatalf("expected default requests per minute > 0")
	}
	if cfg.RateLimiting.WindowSeconds <= 0 {
		t.Fatalf("expected default window seconds > 0")
	}
	if !cfg.RateLimiting.BackoffEnabled {
		t.Fatalf("expected backoff enabled by default")
	}
	if cfg.RateLimiting.BackoffBaseSeconds <= 0 {
		t.Fatalf("expected default backoff base seconds > 0")
	}
	if cfg.RateLimiting.BackoffMaxSeconds <= 0 {
		t.Fatalf("expected default backoff max seconds > 0")
	}
	if cfg.Tenancy.WorkspaceRoot == "" {
		t.Fatalf("expected default workspace root set")
	}
	if cfg.Tenancy.MaxWorkspaceBytes != 0 {
		t.Fatalf(
			"expected default max workspace bytes 0, got %d",
			cfg.Tenancy.MaxWorkspaceBytes,
		)
	}
	if cfg.Tenancy.MaxRequestCount != 0 {
		t.Fatalf(
			"expected default max request count 0, got %d",
			cfg.Tenancy.MaxRequestCount,
		)
	}
	if cfg.Tenancy.MaxConcurrentRequests <= 0 {
		t.Fatalf("expected default max concurrent requests > 0")
	}
	if cfg.Tenancy.MaxSessions <= 0 {
		t.Fatalf("expected default max sessions > 0")
	}
}

func TestValidateConfig(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid local",
			cfg: Config{
				Mode:    "local",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: false,
		},
		{
			name: "valid server with admin token",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath:     t.TempDir(),
					RetentionDays: 7,
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: false,
		},
		{
			name: "valid server without rate limit",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
				},
				RateLimiting: RateLimitConfig{
					Enabled:        false,
					BackoffEnabled: false,
				},
				Tenancy: TenancyConfig{
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 0,
					MaxSessions:           0,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server missing admin token",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server missing token store path",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth:    AuthConfig{AdminToken: "admin-token"},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server missing rpm",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
				},
				RateLimiting: RateLimitConfig{
					Enabled:            true,
					RequestsPerMinute:  0,
					WindowSeconds:      60,
					BackoffEnabled:     true,
					BackoffBaseSeconds: 1,
					BackoffMaxSeconds:  60,
				},
				Tenancy: TenancyConfig{
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server missing window",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
				},
				RateLimiting: RateLimitConfig{
					Enabled:            true,
					RequestsPerMinute:  60,
					WindowSeconds:      0,
					BackoffEnabled:     true,
					BackoffBaseSeconds: 1,
					BackoffMaxSeconds:  60,
				},
				Tenancy: TenancyConfig{
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server backoff base",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
				},
				RateLimiting: RateLimitConfig{
					Enabled:            true,
					RequestsPerMinute:  60,
					WindowSeconds:      60,
					BackoffEnabled:     true,
					BackoffBaseSeconds: 0,
					BackoffMaxSeconds:  60,
				},
				Tenancy: TenancyConfig{
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server backoff max",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
				},
				RateLimiting: RateLimitConfig{
					Enabled:            true,
					RequestsPerMinute:  60,
					WindowSeconds:      60,
					BackoffEnabled:     true,
					BackoffBaseSeconds: 1,
					BackoffMaxSeconds:  0,
				},
				Tenancy: TenancyConfig{
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server backoff max less than base",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
				},
				RateLimiting: RateLimitConfig{
					Enabled:            true,
					RequestsPerMinute:  60,
					WindowSeconds:      60,
					BackoffEnabled:     true,
					BackoffBaseSeconds: 5,
					BackoffMaxSeconds:  1,
				},
				Tenancy: TenancyConfig{
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server tenancy workspace",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot: "",
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server missing tenant store path",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       "",
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server missing audit store path",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server missing usage store path",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server audit retention",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath:     t.TempDir(),
					RetentionDays: -1,
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server negative workspace bytes",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxWorkspaceBytes:     -1,
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server negative request count",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxRequestCount:       -1,
					MaxConcurrentRequests: 5,
					MaxSessions:           2,
				},
				Audit: AuditConfig{
					StorePath: t.TempDir(),
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server tenant limits",
			cfg: Config{
				Mode:    "server",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
				Auth: AuthConfig{
					AdminToken:     "admin-token",
					TokenStorePath: t.TempDir(),
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
					WorkspaceRoot:         t.TempDir(),
					TenantStorePath:       t.TempDir(),
					MaxConcurrentRequests: -1,
					MaxSessions:           -1,
				},
				Usage: UsageConfig{
					StorePath: t.TempDir(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			cfg: Config{
				Mode:    "unknown",
				Address: "127.0.0.1:8080",
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: true,
		},
		{
			name: "missing address",
			cfg: Config{
				Mode:    "local",
				Address: "",
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := Validate(testCase.cfg)
			if testCase.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadConfigWithOverrides(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	configContents := []byte("mode: server\naddress: 0.0.0.0:9000\n")
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("ARCAFLOW_MCP_MODE", "local")
	t.Setenv("ARCAFLOW_MCP_ADDRESS", "127.0.0.1:7777")
	t.Setenv("ARCAFLOW_MCP_LOG_LEVEL", "debug")
	t.Setenv("ARCAFLOW_MCP_ADMIN_TOKEN", "admin-token")
	t.Setenv("ARCAFLOW_MCP_TOKEN_STORE_PATH", filepath.Join(tempDir, "tokens.json"))
	t.Setenv("ARCAFLOW_MCP_RATE_LIMIT_ENABLED", "true")
	t.Setenv("ARCAFLOW_MCP_RATE_LIMIT_RPM", "120")
	t.Setenv("ARCAFLOW_MCP_RATE_LIMIT_WINDOW_SECONDS", "30")
	t.Setenv("ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_ENABLED", "true")
	t.Setenv("ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_BASE_SECONDS", "2")
	t.Setenv("ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_MAX_SECONDS", "10")
	t.Setenv("ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT", tempDir)
	t.Setenv(
		"ARCAFLOW_MCP_TENANT_STORE_PATH",
		filepath.Join(tempDir, "tenants.json"),
	)
	t.Setenv("ARCAFLOW_MCP_TENANT_MAX_WORKSPACE_BYTES", "4096")
	t.Setenv("ARCAFLOW_MCP_TENANT_MAX_REQUESTS", "200")
	t.Setenv("ARCAFLOW_MCP_TENANT_MAX_CONCURRENT", "7")
	t.Setenv("ARCAFLOW_MCP_TENANT_MAX_SESSIONS", "3")
	t.Setenv(
		"ARCAFLOW_MCP_AUDIT_STORE_PATH",
		filepath.Join(tempDir, "audit.json"),
	)
	t.Setenv("ARCAFLOW_MCP_AUDIT_RETENTION_DAYS", "10")
	t.Setenv(
		"ARCAFLOW_MCP_USAGE_STORE_PATH",
		filepath.Join(tempDir, "usage.json"),
	)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Mode != "local" {
		t.Fatalf("expected mode override local, got %q", cfg.Mode)
	}
	if cfg.Address != "127.0.0.1:7777" {
		t.Fatalf("expected address override, got %q", cfg.Address)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected log level override, got %q", cfg.Logging.Level)
	}
	if cfg.Auth.AdminToken != "admin-token" {
		t.Fatalf("expected admin token override, got %q", cfg.Auth.AdminToken)
	}
	if cfg.Auth.TokenStorePath == "" {
		t.Fatalf("expected token store path override")
	}
	if !cfg.RateLimiting.Enabled {
		t.Fatalf("expected rate limiting enabled")
	}
	if cfg.RateLimiting.RequestsPerMinute != 120 {
		t.Fatalf(
			"expected rpm override 120, got %d",
			cfg.RateLimiting.RequestsPerMinute,
		)
	}
	if cfg.RateLimiting.WindowSeconds != 30 {
		t.Fatalf(
			"expected window override 30, got %d",
			cfg.RateLimiting.WindowSeconds,
		)
	}
	if !cfg.RateLimiting.BackoffEnabled {
		t.Fatalf("expected backoff enabled")
	}
	if cfg.RateLimiting.BackoffBaseSeconds != 2 {
		t.Fatalf(
			"expected backoff base override 2, got %d",
			cfg.RateLimiting.BackoffBaseSeconds,
		)
	}
	if cfg.RateLimiting.BackoffMaxSeconds != 10 {
		t.Fatalf(
			"expected backoff max override 10, got %d",
			cfg.RateLimiting.BackoffMaxSeconds,
		)
	}
	if cfg.Tenancy.WorkspaceRoot != tempDir {
		t.Fatalf(
			"expected workspace root override %q, got %q",
			tempDir,
			cfg.Tenancy.WorkspaceRoot,
		)
	}
	if cfg.Tenancy.TenantStorePath == "" {
		t.Fatalf("expected tenant store path override")
	}
	if cfg.Audit.StorePath == "" {
		t.Fatalf("expected audit store path override")
	}
	if cfg.Usage.StorePath == "" {
		t.Fatalf("expected usage store path override")
	}
	if cfg.Tenancy.MaxConcurrentRequests != 7 {
		t.Fatalf(
			"expected max concurrent override 7, got %d",
			cfg.Tenancy.MaxConcurrentRequests,
		)
	}
	if cfg.Tenancy.MaxWorkspaceBytes != 4096 {
		t.Fatalf(
			"expected max workspace bytes 4096, got %d",
			cfg.Tenancy.MaxWorkspaceBytes,
		)
	}
	if cfg.Tenancy.MaxRequestCount != 200 {
		t.Fatalf(
			"expected max requests 200, got %d",
			cfg.Tenancy.MaxRequestCount,
		)
	}
	if cfg.Tenancy.MaxSessions != 3 {
		t.Fatalf(
			"expected max sessions override 3, got %d",
			cfg.Tenancy.MaxSessions,
		)
	}
	if cfg.Audit.RetentionDays != 10 {
		t.Fatalf(
			"expected audit retention 10, got %d",
			cfg.Audit.RetentionDays,
		)
	}
}
