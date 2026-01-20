// Command arcaflow-mcp starts the MCP server entrypoint.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/config"
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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Logging.Level),
	}))
	slog.SetDefault(logger)

	switch cfg.Mode {
	case modeLocal:
		return errors.New("local mode not implemented yet")
	case modeServer:
		return errors.New("server mode not implemented yet")
	default:
		return fmt.Errorf("unknown mode %q", cfg.Mode)
	}
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
