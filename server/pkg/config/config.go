// Package config provides configuration loading and validation.
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for the MCP server.
type Config struct {
	Mode    string        `yaml:"mode"`
	Address string        `yaml:"address"`
	Logging LoggingConfig `yaml:"logging"`
}

// LoggingConfig controls structured logging behavior.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// Default returns a baseline configuration.
func Default() Config {
	return Config{
		Mode:    "local",
		Address: "127.0.0.1:8080",
		Logging: LoggingConfig{Level: "info"},
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

	return nil
}
