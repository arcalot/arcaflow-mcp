package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestChooseLogOutput(t *testing.T) {
	if got := chooseLogOutput(modeLocal); got != os.Stderr {
		t.Fatalf("expected stderr for local mode")
	}
	if got := chooseLogOutput("server"); got != os.Stdout {
		t.Fatalf("expected stdout for server mode")
	}
}

func TestParseLogLevel(t *testing.T) {
	if got := parseLogLevel("debug"); got != slog.LevelDebug {
		t.Fatalf("expected debug level")
	}
	if got := parseLogLevel("warn"); got != slog.LevelWarn {
		t.Fatalf("expected warn level")
	}
	if got := parseLogLevel("error"); got != slog.LevelError {
		t.Fatalf("expected error level")
	}
	if got := parseLogLevel("info"); got != slog.LevelInfo {
		t.Fatalf("expected info level")
	}
	if got := parseLogLevel("unknown"); got != slog.LevelInfo {
		t.Fatalf("expected info level for unknown")
	}
}

func TestRegisterDefaultToolsPing(t *testing.T) {
	server := protocol.NewServer(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		protocol.ServerInfo{Name: "test", Version: "0"},
	)
	registerDefaultTools(server, nil)

	initialize := protocol.Request{
		JSONRPC: "2.0",
		ID:      mustID(t, 1),
		Method:  "initialize",
		Params: mustMarshal(t, protocol.InitializeParams{
			ProtocolVersion: protocol.ProtocolVersion,
		}),
		IDSet: true,
	}
	initialized := protocol.Request{
		JSONRPC: "2.0",
		ID:      mustID(t, 2),
		Method:  "initialized",
		Params:  json.RawMessage(`{}`),
		IDSet:   true,
	}
	ping := protocol.Request{
		JSONRPC: "2.0",
		ID:      mustID(t, 3),
		Method:  "tools/call",
		Params: mustMarshal(t, protocol.ToolsCallParams{
			Name:      "ping",
			Arguments: map[string]interface{}{"message": "hello"},
		}),
		IDSet: true,
	}

	ctx := context.Background()
	if _, err := server.Handle(ctx, mustMarshal(t, initialize)); err != nil {
		t.Fatalf("initialize request failed: %v", err)
	}
	if _, err := server.Handle(ctx, mustMarshal(t, initialized)); err != nil {
		t.Fatalf("initialized request failed: %v", err)
	}
	responses, err := server.Handle(ctx, mustMarshal(t, ping))
	if err != nil {
		t.Fatalf("ping request failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	var response protocol.Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("expected success response, got error %v", response.Error)
	}
}

func TestRunRejectsInvalidMode(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configContent := []byte(`
mode: local
address: "127.0.0.1:9999"
logging:
  level: info
`)
	if err := os.WriteFile(configPath, configContent, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalArgs := os.Args
	originalFlag := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = []string{
		"arcaflow-mcp",
		"-config",
		configPath,
		"-mode",
		"invalid",
	}
	t.Cleanup(func() {
		os.Args = originalArgs
		flag.CommandLine = originalFlag
	})

	if err := run(); err == nil {
		t.Fatalf("expected invalid mode error")
	}
}

func TestRunMissingConfigFile(t *testing.T) {
	resetFlags(t, []string{
		"arcaflow-mcp",
		"-config",
		filepath.Join(t.TempDir(), "missing.yaml"),
	})

	if err := run(); err == nil {
		t.Fatalf("expected missing config error")
	}
}

func TestRunInvalidAddress(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configContent := []byte(`
mode: local
address: ""
logging:
  level: info
`)
	if err := os.WriteFile(configPath, configContent, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags(t, []string{
		"arcaflow-mcp",
		"-config",
		configPath,
	})

	if err := run(); err == nil {
		t.Fatalf("expected invalid address error")
	}
}

func resetFlags(t *testing.T, args []string) {
	t.Helper()
	originalArgs := os.Args
	originalFlag := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = args
	t.Cleanup(func() {
		os.Args = originalArgs
		flag.CommandLine = originalFlag
	})
}

func TestRunServerModeTokenStoreError(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.yaml")
	configContent := []byte(`
mode: server
address: "127.0.0.1:0"
auth:
  admin_token: "admin"
  token_store_path: "tokens.json"
tenancy:
  workspace_root: "` + baseDir + `"
  tenant_store_path: "` + filepath.Join(baseDir, "tenants.json") + `"
audit:
  store_path: "` + filepath.Join(baseDir, "audit.json") + `"
usage:
  store_path: "` + filepath.Join(baseDir, "usage.json") + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags(t, []string{
		"arcaflow-mcp",
		"-config",
		configPath,
		"-mode",
		"server",
	})

	if err := run(); err == nil {
		t.Fatalf("expected token store error")
	}
}

func TestRunServerModeAuditStoreError(t *testing.T) {
	baseDir := t.TempDir()
	badDir := filepath.Join(baseDir, "audit-dir")
	if err := os.WriteFile(badDir, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write bad dir: %v", err)
	}
	tokenDir := filepath.Join(baseDir, "tokens")
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		t.Fatalf("mkdir token dir: %v", err)
	}
	configPath := filepath.Join(baseDir, "config.yaml")
	configContent := []byte(`
mode: server
address: "127.0.0.1:0"
auth:
  admin_token: "admin"
  token_store_path: "` + filepath.Join(tokenDir, "tokens.json") + `"
tenancy:
  workspace_root: "` + baseDir + `"
  tenant_store_path: "` + filepath.Join(baseDir, "tenants.json") + `"
audit:
  store_path: "` + filepath.Join(badDir, "audit.json") + `"
usage:
  store_path: "` + filepath.Join(baseDir, "usage.json") + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags(t, []string{
		"arcaflow-mcp",
		"-config",
		configPath,
		"-mode",
		"server",
	})

	if err := run(); err == nil {
		t.Fatalf("expected audit store error")
	}
}

func TestRunServerModeWorkspaceError(t *testing.T) {
	baseDir := t.TempDir()
	workspaceFile := filepath.Join(baseDir, "workspace-root")
	if err := os.WriteFile(workspaceFile, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}
	tokenDir := filepath.Join(baseDir, "tokens")
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		t.Fatalf("mkdir token dir: %v", err)
	}
	configPath := filepath.Join(baseDir, "config.yaml")
	configContent := []byte(`
mode: server
address: "127.0.0.1:0"
auth:
  admin_token: "admin"
  token_store_path: "` + filepath.Join(tokenDir, "tokens.json") + `"
tenancy:
  workspace_root: "` + workspaceFile + `"
  tenant_store_path: "` + filepath.Join(baseDir, "tenants.json") + `"
audit:
  store_path: "` + filepath.Join(baseDir, "audit.json") + `"
usage:
  store_path: "` + filepath.Join(baseDir, "usage.json") + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags(t, []string{
		"arcaflow-mcp",
		"-config",
		configPath,
		"-mode",
		"server",
	})

	if err := run(); err == nil {
		t.Fatalf("expected workspace manager error")
	}
}

func mustMarshal(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return encoded
}

func mustID(t *testing.T, value int) *json.RawMessage {
	t.Helper()
	raw := mustMarshal(t, value)
	return &raw
}
