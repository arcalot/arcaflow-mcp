// Package version centralizes server version resolution.
package version

import (
	"os"
	"path/filepath"
	"strings"
)

const fallbackVersion = "dev"

// Current returns the runtime server version.
func Current() string {
	if value := strings.TrimSpace(os.Getenv("ARCAFLOW_MCP_VERSION")); value != "" {
		return value
	}

	if value := readVersion("VERSION"); value != "" {
		return value
	}

	exe, err := os.Executable()
	if err == nil {
		base := filepath.Dir(exe)
		if value := readVersion(filepath.Join(base, "..", "VERSION")); value != "" {
			return value
		}
	}

	return fallbackVersion
}

func readVersion(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}
