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
}
