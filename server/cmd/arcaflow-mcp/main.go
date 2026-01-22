// Command arcaflow-mcp starts the MCP server entrypoint.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/audit"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/config"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/ratelimit"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
	"github.com/arcalot/arcaflow-mcp/server/pkg/transport/httpserver"
	"github.com/arcalot/arcaflow-mcp/server/pkg/transport/stdio"
	"github.com/arcalot/arcaflow-mcp/server/pkg/version"
)

const (
	modeLocal  = "local"
	modeServer = "server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("arcaflow-mcp failed", "error", err)
		os.Exit(1)
	}
}

// run wires flags and selects the initial transport mode.
func run() error {
	configPath := flag.String("config", "", "path to YAML config file")
	mode := flag.String("mode", "", "runtime mode: local or server")
	address := flag.String("address", "", "listen address (host:port)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *mode != "" {
		cfg.Mode = *mode
	}
	if *address != "" {
		cfg.Address = *address
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	logOutput := chooseLogOutput(cfg.Mode)

	logger := slog.New(slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Logging.Level),
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	switch cfg.Mode {
	case modeLocal:
		serverVersion := version.Current()
		handler := protocol.NewServer(
			logger.With("component", "protocol"),
			protocol.ServerInfo{
				Name:    "arcaflow-mcp",
				Version: serverVersion,
			},
		)
		registerDefaultTools(handler)
		server := stdio.NewServer(
			handler,
			os.Stdin,
			os.Stdout,
			logger.With("component", "stdio"),
		)

		slog.Info("starting local stdio server")
		return server.Serve(ctx)
	case modeServer:
		serverVersion := version.Current()
		handler := protocol.NewServer(
			logger.With("component", "protocol"),
			protocol.ServerInfo{
				Name:    "arcaflow-mcp",
				Version: serverVersion,
			},
		)
		registerDefaultTools(handler)
		tokenStore, err := auth.NewFileStore(cfg.Auth.TokenStorePath)
		if err != nil {
			return err
		}
		authManager, err := auth.NewManager(cfg.Auth.AdminToken, tokenStore)
		if err != nil {
			return err
		}
		auditStore, err := audit.NewFileStore(
			cfg.Audit.StorePath,
			cfg.Audit.RetentionDays,
		)
		if err != nil {
			return err
		}
		workspaceManager, err := tenant.NewWorkspaceManager(
			cfg.Tenancy.WorkspaceRoot,
		)
		if err != nil {
			return err
		}
		tenantStore, err := tenant.NewFileStore(cfg.Tenancy.TenantStorePath)
		if err != nil {
			return err
		}
		usageStore, err := tenant.NewFileUsageStore(cfg.Usage.StorePath)
		if err != nil {
			return err
		}
		requestLimiter := tenant.NewLimiter(cfg.Tenancy.MaxConcurrentRequests)
		sessionLimiter := tenant.NewLimiter(cfg.Tenancy.MaxSessions)
		var limiter *ratelimit.Limiter
		if cfg.RateLimiting.Enabled {
			limiter, err = ratelimit.NewLimiter(ratelimit.Config{
				Limit:          cfg.RateLimiting.RequestsPerMinute,
				Window:         time.Duration(cfg.RateLimiting.WindowSeconds) * time.Second,
				BackoffEnabled: cfg.RateLimiting.BackoffEnabled,
				BackoffBase: time.Duration(
					cfg.RateLimiting.BackoffBaseSeconds,
				) * time.Second,
				BackoffMax: time.Duration(
					cfg.RateLimiting.BackoffMaxSeconds,
				) * time.Second,
			})
			if err != nil {
				return err
			}
		}
		var analysisClient *analysis.Client
		if cfg.Analysis.HTTPURL != "" {
			analysisClient = analysis.NewClient(cfg.Analysis.HTTPURL)
			logger.Info("analysis client configured", "url", cfg.Analysis.HTTPURL)
		}
		httpServer := httpserver.NewServer(
			httpserver.Config{
				Address:          cfg.Address,
				AuthManager:      authManager,
				AnalysisClient:   analysisClient,
				RateLimiter:      limiter,
				WorkspaceManager: workspaceManager,
				RequestLimiter:   requestLimiter,
				SessionLimiter:   sessionLimiter,
				TenantStore:      tenantStore,
				UsageStore:       usageStore,
				AuditStore:       auditStore,
				Quota: tenant.Quota{
					MaxWorkspaceBytes: cfg.Tenancy.MaxWorkspaceBytes,
					MaxRequests:       cfg.Tenancy.MaxRequestCount,
				},
			},
			handler,
			logger.With("component", "http"),
		)
		slog.Info("starting server mode", "address", cfg.Address)
		return httpServer.Serve(ctx)
	default:
		return fmt.Errorf("unknown mode %q", cfg.Mode)
	}
}

func registerDefaultTools(server *protocol.Server) {
	if server == nil {
		return
	}
	server.RegisterTool(protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "ping",
			Description: "Ping the server to verify connectivity.",
			InputSchema: json.RawMessage(
				`{"type":"object","properties":{"message":{"type":"string"}},"additionalProperties":false}`,
			),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			_ = ctx
			message := "pong"
			if value, ok := arguments["message"].(string); ok && value != "" {
				message = value
			}
			return protocol.ToolsCallResult{
				Content: []protocol.ToolContent{
					{
						Type: "text",
						Text: message,
					},
				},
			}, nil
		},
	})
}

func chooseLogOutput(mode string) io.Writer {
	// Use stderr in stdio mode so protocol output stays clean.
	logOutput := io.Writer(os.Stdout)
	if mode == modeLocal {
		logOutput = os.Stderr
	}

	return logOutput
}

// parseLogLevel maps string levels to slog levels.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
