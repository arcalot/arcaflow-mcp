package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"gopkg.in/yaml.v3"
)

const (
	executionScheme    = "execution"
	executionLogScheme = "execution-log"
)

// ExecutionResourceProvider exposes execution result resources.
type ExecutionResourceProvider struct {
	logger *slog.Logger
	mu     sync.RWMutex
	cache  map[string]map[string]protocol.ResourceItem
}

// NewExecutionResourceProvider constructs an execution resource provider.
func NewExecutionResourceProvider(logger *slog.Logger) *ExecutionResourceProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExecutionResourceProvider{
		logger: logger,
		cache:  make(map[string]map[string]protocol.ResourceItem),
	}
}

// List returns cached resources for the current tenant.
func (provider *ExecutionResourceProvider) List(
	ctx context.Context,
) ([]protocol.ResourceItem, *protocol.ErrorObject) {
	tenantID := tenantFromContext(ctx)
	provider.mu.RLock()
	defer provider.mu.RUnlock()
	items := provider.cache[tenantID]
	if len(items) == 0 {
		return []protocol.ResourceItem{}, nil
	}
	result := make([]protocol.ResourceItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].URI < result[j].URI
	})
	return result, nil
}

// Read resolves execution result resources from URI metadata.
func (provider *ExecutionResourceProvider) Read(
	ctx context.Context,
	uri string,
) (*protocol.ResourceContent, bool, *protocol.ErrorObject) {
	params, ok, err := parseExecutionResourceURI(uri)
	if !ok {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			"invalid execution resource uri",
			map[string]string{"error": err.Error()},
		)
	}

	rawText, sizeBytes, shaSum, err := loadResultContent(
		ctx,
		params.SourceKind,
		params.Location,
	)
	if err != nil {
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			"execution result load failed",
			map[string]string{"error": err.Error()},
		)
	}

	formatHint := params.FormatHint
	if params.ForceLog {
		formatHint = "log"
		params.IncludeRaw = true
	}
	format, payloadValue, err := parseResultPayload(rawText, formatHint)
	if err != nil {
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			"execution result parse failed",
			map[string]string{"error": err.Error()},
		)
	}

	result := executionResourcePayload{
		Source: executionSource{
			Kind:     params.SourceKind,
			Location: params.Location,
		},
		Format:        format,
		Payload:       payloadValue,
		SizeBytes:     sizeBytes,
		ContentSHA256: shaSum,
	}
	if params.IncludeRaw {
		result.RawText = rawText
	}

	content, errObj := provider.renderResource(uri, result)
	if errObj != nil {
		return nil, true, errObj
	}
	cacheName := executionScheme
	if params.ForceLog {
		cacheName = executionLogScheme
	}
	provider.cacheResource(ctx, uri, cacheName, "Execution result")
	return content, true, nil
}

type executionResourceParams struct {
	SourceKind string
	Location   string
	FormatHint string
	IncludeRaw bool
	ForceLog   bool
}

type executionSource struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
}

type executionResourcePayload struct {
	Source        executionSource `json:"source"`
	Format        string          `json:"format"`
	Payload       interface{}     `json:"payload"`
	RawText       string          `json:"raw_text,omitempty"`
	SizeBytes     int64           `json:"size_bytes,omitempty"`
	ContentSHA256 string          `json:"content_sha256,omitempty"`
}

func parseExecutionResourceURI(
	rawURI string,
) (executionResourceParams, bool, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return executionResourceParams{}, false, err
	}
	if parsed.Scheme != executionScheme && parsed.Scheme != executionLogScheme {
		return executionResourceParams{}, false, nil
	}
	query := parsed.Query()
	kind := strings.TrimSpace(parsed.Host)
	if kind == "" {
		kind = strings.TrimSpace(parsed.Opaque)
	}
	if kind == "" {
		kind = strings.TrimSpace(query.Get("kind"))
	}
	location := strings.TrimSpace(query.Get("location"))
	if location == "" && strings.TrimSpace(parsed.Path) != "" {
		location = strings.TrimPrefix(parsed.Path, "/")
	}
	if kind == "" {
		return executionResourceParams{}, true, fmt.Errorf("source kind required")
	}
	if location == "" {
		return executionResourceParams{}, true, fmt.Errorf("source location required")
	}
	includeRaw := true
	if rawFlag := strings.TrimSpace(query.Get("include_raw")); rawFlag != "" {
		parsedFlag, err := strconv.ParseBool(rawFlag)
		if err != nil {
			return executionResourceParams{}, true, fmt.Errorf(
				"include_raw must be boolean",
			)
		}
		includeRaw = parsedFlag
	}
	return executionResourceParams{
		SourceKind: strings.ToLower(kind),
		Location:   location,
		FormatHint: strings.TrimSpace(query.Get("format")),
		IncludeRaw: includeRaw,
		ForceLog:   parsed.Scheme == executionLogScheme,
	}, true, nil
}

func (provider *ExecutionResourceProvider) renderResource(
	uri string,
	payload interface{},
) (*protocol.ResourceContent, *protocol.ErrorObject) {
	raw, err := json.Marshal(payload)
	if err != nil {
		if provider.logger != nil {
			provider.logger.Error("marshal resource payload", "error", err)
		}
		return nil, resourceError(
			protocol.ErrInternal,
			"failed to encode resource payload",
			map[string]string{"error": err.Error()},
		)
	}
	content := protocol.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(raw),
	}
	return &content, nil
}

func (provider *ExecutionResourceProvider) cacheResource(
	ctx context.Context,
	uri string,
	name string,
	description string,
) {
	tenantID := tenantFromContext(ctx)
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if _, ok := provider.cache[tenantID]; !ok {
		provider.cache[tenantID] = make(map[string]protocol.ResourceItem)
	}
	provider.cache[tenantID][uri] = protocol.ResourceItem{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    "application/json",
	}
}

func loadResultContent(
	ctx context.Context,
	kind string,
	location string,
) (string, int64, string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "filesystem":
		path := filepath.Clean(location)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", 0, "", fmt.Errorf("read result file: %w", err)
		}
		sum := sha256.Sum256(data)
		return string(data), int64(len(data)), hex.EncodeToString(sum[:]), nil
	case "url":
		if _, err := url.Parse(location); err != nil {
			return "", 0, "", fmt.Errorf("parse url: %w", err)
		}
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return "", 0, "", fmt.Errorf("build url request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", 0, "", fmt.Errorf("fetch url: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				slog.Default().Warn("close result response", "error", err)
			}
		}()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
			return "", 0, "", fmt.Errorf("url status %d", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", 0, "", fmt.Errorf("read url body: %w", err)
		}
		sum := sha256.Sum256(body)
		return string(body), int64(len(body)), hex.EncodeToString(sum[:]), nil
	default:
		return "", 0, "", fmt.Errorf("unsupported source kind %q", kind)
	}
}

func parseResultPayload(
	rawText string,
	formatHint string,
) (string, interface{}, error) {
	formatHint = strings.ToLower(strings.TrimSpace(formatHint))
	switch formatHint {
	case "json":
		var payload interface{}
		if err := json.Unmarshal([]byte(rawText), &payload); err != nil {
			return "", nil, err
		}
		return "json", payload, nil
	case "yaml", "yml":
		var payload interface{}
		if err := yaml.Unmarshal([]byte(rawText), &payload); err != nil {
			return "", nil, err
		}
		return "yaml", payload, nil
	case "log", "txt":
		return "log", rawText, nil
	case "":
		if payload, ok := tryParseJSON(rawText); ok {
			return "json", payload, nil
		}
		if payload, ok := tryParseYAML(rawText); ok {
			return "yaml", payload, nil
		}
		return "log", rawText, nil
	default:
		return "", nil, fmt.Errorf("unsupported format hint %q", formatHint)
	}
}

func tryParseJSON(rawText string) (interface{}, bool) {
	var payload interface{}
	if err := json.Unmarshal([]byte(rawText), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func tryParseYAML(rawText string) (interface{}, bool) {
	var payload interface{}
	if err := yaml.Unmarshal([]byte(rawText), &payload); err != nil {
		return nil, false
	}
	return payload, true
}
