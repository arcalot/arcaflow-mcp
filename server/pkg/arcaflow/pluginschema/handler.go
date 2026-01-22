package pluginschema

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaEntry stores a loaded plugin schema.
type SchemaEntry struct {
	StepID   string
	Location string
	Schema   json.RawMessage
	LoadedAt time.Time
}

// Handler loads and caches plugin schemas referenced by workflows.
type Handler struct {
	logger     *slog.Logger
	httpClient *http.Client
	cache      map[string]SchemaEntry
}

// NewHandler constructs a plugin schema handler.
func NewHandler(logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		logger:     logger,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cache:      make(map[string]SchemaEntry),
	}
}

// LoadFromWorkflow loads plugin schemas referenced by a workflow document.
func (handler *Handler) LoadFromWorkflow(
	ctx context.Context,
	workflowContent []byte,
	workflowPath string,
) ([]SchemaEntry, error) {
	root, err := parseWorkflowDocument(workflowContent)
	if err != nil {
		return nil, err
	}

	stepsRaw, ok := root["steps"]
	if !ok {
		return nil, nil
	}
	steps, ok := stepsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow steps must be an object")
	}

	var entries []SchemaEntry
	for stepID, stepRaw := range steps {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			continue
		}
		locations := extractSchemaLocations(stepMap)
		for _, location := range locations {
			entry, err := handler.loadSchema(ctx, stepID, location, workflowPath)
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (handler *Handler) loadSchema(
	ctx context.Context,
	stepID string,
	location string,
	workflowPath string,
) (SchemaEntry, error) {
	if ctx.Err() != nil {
		return SchemaEntry{}, ctx.Err()
	}
	cacheKey := stepID + "|" + location
	if cached, ok := handler.cache[cacheKey]; ok {
		return cached, nil
	}

	payload, err := handler.readSchema(location, workflowPath)
	if err != nil {
		return SchemaEntry{}, err
	}
	entry := SchemaEntry{
		StepID:   stepID,
		Location: location,
		Schema:   payload,
		LoadedAt: time.Now().UTC(),
	}
	handler.cache[cacheKey] = entry
	return entry, nil
}

func (handler *Handler) readSchema(
	location string,
	workflowPath string,
) (json.RawMessage, error) {
	if location == "" {
		return nil, fmt.Errorf("plugin schema location is empty")
	}
	if isHTTPURL(location) {
		req, err := http.NewRequest(http.MethodGet, location, nil)
		if err != nil {
			return nil, fmt.Errorf("build schema request: %w", err)
		}
		resp, err := handler.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch schema: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				handler.logger.Warn(
					"close schema response",
					"error",
					err,
				)
			}
		}()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("schema fetch status %d", resp.StatusCode)
		}
		return parseSchemaPayload(resp.Body)
	}

	path := location
	if !filepath.IsAbs(path) {
		if workflowPath == "" {
			return nil, fmt.Errorf("relative schema path requires workflow path")
		}
		path = filepath.Join(filepath.Dir(workflowPath), location)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema file: %w", err)
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read schema file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close schema file: %w", err)
	}
	return parseSchemaPayload(bytes.NewReader(raw))
}

// Cache returns a copy of the schema cache.
func (handler *Handler) Cache() map[string]SchemaEntry {
	copy := make(map[string]SchemaEntry, len(handler.cache))
	for key, value := range handler.cache {
		copy[key] = value
	}
	return copy
}

func extractSchemaLocations(step map[string]interface{}) []string {
	var locations []string
	for _, key := range []string{"plugin_schema_ref", "plugin_schema"} {
		if value, ok := step[key].(string); ok && value != "" {
			locations = append(locations, value)
		}
	}
	if plugin, ok := step["plugin"].(map[string]interface{}); ok {
		for _, key := range []string{"schema_ref", "schema"} {
			if value, ok := plugin[key].(string); ok && value != "" {
				locations = append(locations, value)
			}
		}
	}
	return locations
}

func parseSchemaPayload(reader io.Reader) (json.RawMessage, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read schema payload: %w", err)
	}
	if json.Valid(raw) {
		return raw, nil
	}
	var payload interface{}
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse schema yaml: %w", err)
	}
	normalized := normalizeYAML(payload)
	return json.Marshal(normalized)
}

func parseWorkflowDocument(content []byte) (map[string]interface{}, error) {
	if json.Valid(content) {
		var payload interface{}
		if err := json.Unmarshal(content, &payload); err != nil {
			return nil, fmt.Errorf("parse workflow json: %w", err)
		}
		return ensureObject(payload)
	}
	var payload interface{}
	if err := yaml.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("parse workflow yaml: %w", err)
	}
	return ensureObject(normalizeYAML(payload))
}

func ensureObject(payload interface{}) (map[string]interface{}, error) {
	root, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow document must be a JSON object")
	}
	return root, nil
}

func normalizeYAML(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			normalized[key] = normalizeYAML(child)
		}
		return normalized
	case map[interface{}]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			normalized[fmt.Sprint(key)] = normalizeYAML(child)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, 0, len(typed))
		for _, child := range typed {
			normalized = append(normalized, normalizeYAML(child))
		}
		return normalized
	default:
		return typed
	}
}

func isHTTPURL(location string) bool {
	parsed, err := url.Parse(location)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
