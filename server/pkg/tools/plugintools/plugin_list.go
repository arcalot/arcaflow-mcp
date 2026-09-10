package plugintools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/pluginmeta"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/quay"
)

// defaultOrgs are the Quay organisations scanned for
// Arcaflow plugins when no custom list is provided.
var defaultOrgs = []string{
	"arcalot",
	"redhat-performance",
}

// defaultCacheTTL is the cache lifetime used when the
// caller passes zero.
const defaultCacheTTL = 1 * time.Hour

// pluginPrefix is stripped from repo names when
// generating a human-readable description fallback.
const pluginPrefix = "arcaflow-plugin-"

// PluginCatalogService provides cached access to the
// Arcaflow plugin catalog from Quay.io registries.
type PluginCatalogService struct {
	quayClient *quay.Client
	meta       *pluginmeta.Catalog
	logger     *slog.Logger
	orgs       []string

	cacheMu   sync.RWMutex
	cache     []PluginInfo
	cacheTime time.Time
	cacheTTL  time.Duration

	// fetchMu ensures only one Quay fetch runs at
	// a time, preventing thundering-herd requests.
	fetchMu sync.Mutex
}

// PluginInfo holds enriched metadata about a plugin.
type PluginInfo struct {
	Name          string   `json:"name"`
	Image         string   `json:"image"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Keywords      []string `json:"keywords"`
	Architectures []string `json:"architectures"`
	Category      string   `json:"category"`
	DefaultStep   string   `json:"default_step"`
	Steps         []string `json:"steps"`
}

// NewPluginCatalogService creates a catalog service
// that discovers plugins from the given Quay
// organisations and enriches them with metadata.
// A zero cacheTTL defaults to one hour.
func NewPluginCatalogService(
	quayClient *quay.Client,
	meta *pluginmeta.Catalog,
	logger *slog.Logger,
	cacheTTL time.Duration,
) *PluginCatalogService {
	if logger == nil {
		logger = slog.Default()
	}
	if cacheTTL == 0 {
		cacheTTL = defaultCacheTTL
	}
	return &PluginCatalogService{
		quayClient: quayClient,
		meta:       meta,
		logger:     logger,
		orgs:       defaultOrgs,
		cacheTTL:   cacheTTL,
	}
}

// ListPlugins returns the full plugin catalog, using
// the cache when it is still valid. On Quay errors
// it falls back to stale cache data or an empty list.
func (s *PluginCatalogService) ListPlugins(
	ctx context.Context,
) ([]PluginInfo, error) {
	// Fast path: cache is valid.
	s.cacheMu.RLock()
	if len(s.cache) > 0 &&
		time.Since(s.cacheTime) < s.cacheTTL {
		result := s.cache
		s.cacheMu.RUnlock()
		return result, nil
	}
	s.cacheMu.RUnlock()

	// Serialize fetches so only one goroutine hits
	// Quay at a time.
	s.fetchMu.Lock()
	defer s.fetchMu.Unlock()

	// Double-check: another goroutine may have
	// refreshed the cache while we waited.
	s.cacheMu.RLock()
	if len(s.cache) > 0 &&
		time.Since(s.cacheTime) < s.cacheTTL {
		result := s.cache
		s.cacheMu.RUnlock()
		return result, nil
	}
	s.cacheMu.RUnlock()

	plugins, err := s.fetchAll(ctx)
	if err != nil {
		s.logger.Warn(
			"quay fetch failed, using fallback",
			"error", err,
		)
		// Return stale cache if available.
		s.cacheMu.RLock()
		stale := s.cache
		s.cacheMu.RUnlock()
		if len(stale) > 0 {
			return stale, nil
		}
		return []PluginInfo{}, nil
	}

	s.cacheMu.Lock()
	s.cache = plugins
	s.cacheTime = time.Now()
	s.cacheMu.Unlock()

	return plugins, nil
}

// fetchAll queries every configured org and builds an
// enriched PluginInfo slice.
func (s *PluginCatalogService) fetchAll(
	ctx context.Context,
) ([]PluginInfo, error) {
	var plugins []PluginInfo
	var lastErr error

	for _, org := range s.orgs {
		repos, err := s.quayClient.ListRepos(ctx, org)
		if err != nil {
			s.logger.Warn(
				"list repos failed",
				"org", org, "error", err,
			)
			lastErr = err
			continue
		}
		for _, repo := range repos {
			info, err := s.buildInfo(ctx, org, repo)
			if err != nil {
				s.logger.Warn(
					"build info failed",
					"repo", repo.Name, "error", err,
				)
				continue
			}
			plugins = append(plugins, info)
		}
	}

	// If we got at least some plugins, treat partial
	// failures as acceptable.
	if len(plugins) > 0 {
		return plugins, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return []PluginInfo{}, nil
}

// buildInfo fetches tags for a single repo, finds the
// latest semver, and enriches with pluginmeta.
func (s *PluginCatalogService) buildInfo(
	ctx context.Context,
	org string,
	repo quay.Repository,
) (PluginInfo, error) {
	tags, err := s.quayClient.ListTags(
		ctx, org, repo.Name,
	)
	if err != nil {
		return PluginInfo{}, fmt.Errorf(
			"list tags for %s/%s: %w",
			org, repo.Name, err,
		)
	}

	version := s.quayClient.LatestSemver(tags)

	entry := s.meta.Lookup(repo.Name)

	desc := repo.Description
	if desc == "" {
		short := strings.TrimPrefix(
			repo.Name, pluginPrefix,
		)
		desc = "Arcaflow plugin: " + short
	}

	return PluginInfo{
		Name:          repo.Name,
		Image:         fmt.Sprintf("quay.io/%s/%s", org, repo.Name),
		Version:       version,
		Description:   desc,
		Keywords:      entry.Keywords,
		Architectures: entry.Architectures,
		Category:      entry.Category,
		DefaultStep:   entry.DefaultStep,
		Steps:         entry.Steps,
	}, nil
}

// pluginListInputSchema is the JSON Schema for the
// plugin_list tool input.
const pluginListInputSchema = `{
  "type": "object",
  "properties": {
    "category": {
      "type": "string",
      "description": "Optional filter by category (e.g., 'storage', 'cpu', 'network', 'stress'). Returns all if omitted."
    },
    "architecture": {
      "type": "string",
      "description": "Optional filter by supported architecture (e.g., 'amd64', 'arm64'). Returns all if omitted."
    }
  },
  "additionalProperties": false
}`

// PluginListResult is the plugin_list tool output.
type PluginListResult struct {
	Plugins  []PluginInfo `json:"plugins"`
	Total    int          `json:"total"`
	Filtered int          `json:"filtered"`
}

// NewPluginListTool registers the plugin_list MCP tool.
func NewPluginListTool(
	service *PluginCatalogService,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "plugin_list",
			Description: "List available Arcaflow " +
				"plugins with metadata. Returns " +
				"plugin names, images, versions, " +
				"keywords, and categories. Use " +
				"category or architecture filters " +
				"to narrow results. Call this first " +
				"to discover what plugins are " +
				"available before using " +
				"plugin_describe for details.",
			InputSchema: json.RawMessage(
				pluginListInputSchema,
			),
		},
		Handler: handlePluginList(service, logger),
	}
}

// handlePluginList returns the handler closure for the
// plugin_list tool.
func handlePluginList(
	service *PluginCatalogService,
	logger *slog.Logger,
) protocol.ToolHandler {
	return func(
		ctx context.Context,
		arguments map[string]interface{},
	) (protocol.ToolsCallResult, *protocol.ErrorObject) {
		category, _ := arguments["category"].(string)
		arch, _ := arguments["architecture"].(string)

		all, err := service.ListPlugins(ctx)
		if err != nil {
			return protocol.ToolsCallResult{}, toolError(
				protocol.ErrInternal,
				"failed to list plugins",
				map[string]string{
					"error": err.Error(),
				},
			)
		}

		total := len(all)
		filtered := filterPlugins(all, category, arch)

		result := PluginListResult{
			Plugins:  filtered,
			Total:    total,
			Filtered: len(filtered),
		}

		return renderJSONResult(result, logger)
	}
}

// filterPlugins applies optional category and
// architecture filters. Both comparisons are
// case-insensitive.
func filterPlugins(
	plugins []PluginInfo,
	category, arch string,
) []PluginInfo {
	if category == "" && arch == "" {
		return plugins
	}

	catLower := strings.ToLower(category)
	archLower := strings.ToLower(arch)

	result := make([]PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		if category != "" &&
			strings.ToLower(p.Category) != catLower {
			continue
		}
		if arch != "" && !hasArch(p, archLower) {
			continue
		}
		result = append(result, p)
	}
	return result
}

// hasArch checks whether a plugin supports the given
// architecture (case-insensitive).
func hasArch(p PluginInfo, arch string) bool {
	for _, a := range p.Architectures {
		if strings.ToLower(a) == arch {
			return true
		}
	}
	return false
}
