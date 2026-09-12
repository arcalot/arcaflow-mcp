package plugintools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"gopkg.in/yaml.v3"
)

// allowedRegistryPrefixes are the image reference
// prefixes we trust enough to execute locally. Images
// from other registries are rejected to prevent running
// arbitrary containers.
var allowedRegistryPrefixes = []string{
	"quay.io/arcalot/",
	"quay.io/redhat-performance/",
}

// SchemaProvider retrieves plugin schemas from container
// images. This interface enables testing with mocks
// instead of requiring a real container runtime.
type SchemaProvider interface {
	HasRuntime() bool
	FullSchema(
		ctx context.Context,
		image string,
	) ([]byte, error)
}

// PluginDescribeResult is the plugin_describe tool
// output payload.
type PluginDescribeResult struct {
	Name             string              `json:"name"`
	Image            string              `json:"image"`
	Description      string              `json:"description"`
	Architectures    []string            `json:"architectures"`
	Steps            map[string]StepInfo `json:"steps"`
	DefaultStep      string              `json:"default_step"`
	SchemasAvailable bool                `json:"schemas_available"`
	SchemaRaw        interface{}         `json:"schema_raw,omitempty"`
}

// StepInfo holds per-step schema information extracted
// from the Arcaflow plugin schema.
type StepInfo struct {
	ID          string       `json:"id"`
	Display     *StepDisplay `json:"display,omitempty"`
	InputSchema interface{}  `json:"input_schema,omitempty"`
	Outputs     interface{}  `json:"outputs,omitempty"`
}

// StepDisplay holds optional display metadata for a
// step, such as a human-readable name and description.
type StepDisplay struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// schemaCache stores retrieved schemas keyed by
// image:version. Tagged versions are cached
// indefinitely; "latest" entries are not cached.
type schemaCache struct {
	mu      sync.RWMutex
	entries map[string]schemaCacheEntry
}

type schemaCacheEntry struct {
	result PluginDescribeResult
}

func newSchemaCache() *schemaCache {
	return &schemaCache{
		entries: make(map[string]schemaCacheEntry),
	}
}

func (c *schemaCache) get(
	key string,
) (PluginDescribeResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok {
		return PluginDescribeResult{}, false
	}
	return e.result, true
}

func (c *schemaCache) set(
	key string,
	result PluginDescribeResult,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = schemaCacheEntry{result: result}
}

// resolveImageRef converts a plugin name or full image
// reference into a validated, fully-qualified image ref
// (without tag). Security: only images from allowed
// registries are accepted.
//
// Accepted inputs:
//   - "arcaflow-plugin-fio"
//     → "quay.io/arcalot/arcaflow-plugin-fio"
//   - "quay.io/arcalot/arcaflow-plugin-fio"
//     → as-is
//   - "quay.io/redhat-performance/arcaflow-plugin-fio"
//     → as-is
//   - "evil.io/malware" → error
func resolveImageRef(plugin string) (string, error) {
	// Strip any tag suffix — version is handled
	// separately.
	if idx := strings.LastIndex(plugin, ":"); idx > 0 {
		plugin = plugin[:idx]
	}

	// Already a full reference from an allowed registry.
	if isAllowedImage(plugin) {
		return plugin, nil
	}

	// Bare plugin name — default to arcalot org.
	if strings.HasPrefix(plugin, "arcaflow-plugin-") {
		return "quay.io/arcalot/" + plugin, nil
	}

	return "", fmt.Errorf(
		"image %q not from an allowed registry; "+
			"accepted prefixes: %v",
		plugin,
		allowedRegistryPrefixes,
	)
}

// isAllowedImage checks if an image reference starts
// with one of the allowed registry+org prefixes.
func isAllowedImage(image string) bool {
	for _, prefix := range allowedRegistryPrefixes {
		if strings.HasPrefix(image, prefix) {
			return true
		}
	}
	return false
}

// parseSchemaOutput parses the raw YAML output from a
// plugin's --schema flag. The Go SDK wraps the schema
// with a "serialized_schema:" prefix, which YAML
// parses as a map with that key. If the prefix is
// absent, the output is treated as the schema itself.
func parseSchemaOutput(
	raw []byte,
) (map[string]interface{}, error) {
	var top map[string]interface{}
	if err := yaml.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	// The Go SDK wraps: "serialized_schema: <yaml>"
	if inner, ok := top["serialized_schema"]; ok {
		if m, ok := inner.(map[string]interface{}); ok {
			return m, nil
		}
	}
	// Direct schema (Python SDK or future formats).
	return top, nil
}

// extractSteps pulls step information out of a parsed
// Arcaflow plugin schema map.
func extractSteps(
	schema map[string]interface{},
) map[string]StepInfo {
	steps := make(map[string]StepInfo)
	stepsRaw, ok := schema["steps"]
	if !ok {
		return steps
	}
	stepsMap, ok := stepsRaw.(map[string]interface{})
	if !ok {
		return steps
	}
	for id, stepRaw := range stepsMap {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			continue
		}
		info := StepInfo{ID: id}
		if display, ok := stepMap["display"]; ok {
			if dm, ok :=
				display.(map[string]interface{}); ok {
				d := &StepDisplay{}
				if n, ok := dm["name"].(string); ok {
					d.Name = n
				}
				if desc, ok :=
					dm["description"].(string); ok {
					d.Description = desc
				}
				info.Display = d
			}
		}
		if input, ok := stepMap["input"]; ok {
			info.InputSchema = input
		}
		if outputs, ok := stepMap["outputs"]; ok {
			info.Outputs = outputs
		}
		steps[id] = info
	}
	return steps
}

// pluginDescribeInputSchema is the JSON Schema for the
// plugin_describe tool input.
const pluginDescribeInputSchema = `{
  "type": "object",
  "properties": {
    "plugin": {
      "type": "string",
      "description": "Plugin name (e.g., 'arcaflow-plugin-fio') or full image reference (e.g., 'quay.io/arcalot/arcaflow-plugin-fio')."
    },
    "version": {
      "type": "string",
      "description": "Optional version tag. Uses 'latest' if omitted."
    }
  },
  "required": ["plugin"],
  "additionalProperties": false
}`

// NewPluginDescribeTool registers the plugin_describe
// MCP tool. It uses the given SchemaProvider to retrieve
// plugin schemas from container images.
func NewPluginDescribeTool(
	provider SchemaProvider,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	cache := newSchemaCache()
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "plugin_describe",
			Description: "Get detailed information " +
				"about a specific Arcaflow plugin " +
				"including its step schemas. " +
				"Returns all steps with their " +
				"input and output schema " +
				"definitions. Use plugin_list " +
				"first to discover available " +
				"plugins, then plugin_describe " +
				"to inspect a specific plugin's " +
				"capabilities and input " +
				"requirements.",
			InputSchema: json.RawMessage(
				pluginDescribeInputSchema,
			),
		},
		Handler: handlePluginDescribe(
			provider, cache, logger,
		),
	}
}

// handlePluginDescribe returns the handler closure for
// the plugin_describe tool.
func handlePluginDescribe(
	provider SchemaProvider,
	cache *schemaCache,
	logger *slog.Logger,
) protocol.ToolHandler {
	return func(
		ctx context.Context,
		arguments map[string]interface{},
	) (protocol.ToolsCallResult, *protocol.ErrorObject) {
		pluginArg, _ := arguments["plugin"].(string)
		if pluginArg == "" {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"plugin is required",
					nil,
				)
		}

		version, _ := arguments["version"].(string)
		if version == "" {
			version = "latest"
		}

		imageBase, err := resolveImageRef(pluginArg)
		if err != nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					err.Error(),
					nil,
				)
		}

		imageRef := imageBase + ":" + version
		repoName := imageBase[strings.LastIndex(
			imageBase, "/")+1:]

		// Cache lookup — only cache tagged versions,
		// not "latest".
		cacheKey := imageRef
		if version != "latest" {
			if cached, ok := cache.get(cacheKey); ok {
				return renderJSONResult(
					cached, logger,
				)
			}
		}

		// No container runtime → return metadata only.
		if provider == nil || !provider.HasRuntime() {
			result := PluginDescribeResult{
				Name:             repoName,
				Image:            imageRef,
				Description:      descFromName(repoName),
				Architectures:    []string{"unknown"},
				Steps:            map[string]StepInfo{},
				SchemasAvailable: false,
			}
			return renderJSONResult(result, logger)
		}

		raw, err := provider.FullSchema(ctx, imageRef)
		if err != nil {
			logger.Warn(
				"schema retrieval failed",
				"image", imageRef,
				"error", err,
			)
			result := PluginDescribeResult{
				Name:             repoName,
				Image:            imageRef,
				Description:      descFromName(repoName),
				Architectures:    []string{"unknown"},
				Steps:            map[string]StepInfo{},
				SchemasAvailable: false,
			}
			return renderJSONResult(result, logger)
		}

		schema, err := parseSchemaOutput(raw)
		if err != nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInternal,
					fmt.Sprintf(
						"failed to parse plugin "+
							"schema: %s",
						err.Error(),
					),
					nil,
				)
		}

		steps := extractSteps(schema)
		defaultStep := ""
		if len(steps) == 1 {
			for id := range steps {
				defaultStep = id
			}
		}

		// Build description from step display metadata
		// when available.
		desc := descFromSteps(repoName, steps)

		result := PluginDescribeResult{
			Name:             repoName,
			Image:            imageRef,
			Description:      desc,
			Architectures:    []string{"unknown"},
			Steps:            steps,
			DefaultStep:      defaultStep,
			SchemasAvailable: true,
			SchemaRaw:        schema,
		}

		// Cache tagged versions indefinitely.
		if version != "latest" {
			cache.set(cacheKey, result)
		}

		return renderJSONResult(result, logger)
	}
}

// descFromName generates a fallback description from a
// plugin repo name.
func descFromName(repoName string) string {
	short := strings.TrimPrefix(
		repoName, "arcaflow-plugin-",
	)
	return "Arcaflow plugin: " + short
}

// descFromSteps tries to build a description from step
// display metadata. If the plugin has a single step
// with a display description, use that. Otherwise fall
// back to the name-based description.
func descFromSteps(
	repoName string,
	steps map[string]StepInfo,
) string {
	if len(steps) == 1 {
		for _, s := range steps {
			if s.Display != nil &&
				s.Display.Description != "" {
				return s.Display.Description
			}
			if s.Display != nil &&
				s.Display.Name != "" {
				return s.Display.Name
			}
		}
	}
	return descFromName(repoName)
}
