// Package pluginmeta provides metadata enrichment for
// Arcaflow plugins. It loads a static YAML config of
// known plugin keywords and categories, and falls back
// to name-based inference for unknown plugins.
package pluginmeta

import (
	_ "embed"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed default_metadata.yaml
var defaultMetadata []byte

// DefaultCatalog returns a Catalog loaded from the
// embedded default metadata. This avoids reliance on
// a file path at runtime.
func DefaultCatalog() *Catalog {
	return NewCatalogFromBytes(defaultMetadata)
}

// Catalog provides plugin metadata for enrichment.
// It uses a two-tier approach: static config for known
// plugins, name-based inference for unknown ones.
type Catalog struct {
	plugins map[string]Entry
}

// Entry holds metadata for a single plugin.
type Entry struct {
	Keywords []string `yaml:"keywords"`
	Category string   `yaml:"category"`
}

// configFile mirrors the YAML structure on disk so we
// can unmarshal the top-level "plugins" map cleanly.
type configFile struct {
	Plugins map[string]Entry `yaml:"plugins"`
}

// pluginPrefix is stripped from repo names before
// splitting into inferred keywords.
const pluginPrefix = "arcaflow-plugin-"

// NewCatalog creates a Catalog from a YAML config file.
// If the file cannot be read or parsed, returns an empty
// catalog that falls back to name-based inference for
// all plugins.
func NewCatalog(configPath string) *Catalog {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &Catalog{plugins: map[string]Entry{}}
	}
	return NewCatalogFromBytes(data)
}

// NewCatalogFromBytes creates a Catalog from raw YAML
// bytes. Useful for testing and embedding. Returns an
// empty catalog on invalid YAML (no panic).
func NewCatalogFromBytes(data []byte) *Catalog {
	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return &Catalog{plugins: map[string]Entry{}}
	}
	if cfg.Plugins == nil {
		cfg.Plugins = map[string]Entry{}
	}
	return &Catalog{plugins: cfg.Plugins}
}

// Lookup returns metadata for a plugin by repo name.
// If the plugin is in the static config, returns that
// entry directly. Otherwise, infers keywords by
// stripping the "arcaflow-plugin-" prefix and splitting
// on hyphens, and sets category to "other".
func (c *Catalog) Lookup(repoName string) Entry {
	if e, ok := c.plugins[repoName]; ok {
		return e
	}
	return inferEntry(repoName)
}

// inferEntry derives keywords from a plugin repo name.
// It removes the common prefix, then splits the
// remainder on hyphens so each segment becomes a keyword.
func inferEntry(repoName string) Entry {
	name := strings.TrimPrefix(repoName, pluginPrefix)
	parts := strings.Split(name, "-")
	// Filter out empty strings that could appear from
	// leading/trailing/double hyphens.
	keywords := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			keywords = append(keywords, p)
		}
	}
	return Entry{
		Keywords: keywords,
		Category: "other",
	}
}
