package workflowtools

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
	"strings"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"gopkg.in/yaml.v3"
)

const workflowResultsLoadInputSchema = `{
  "type": "object",
  "properties": {
    "source": {
      "type": "object",
      "properties": {
        "kind": {
          "type": "string",
          "description": "Result source kind: filesystem or url."
        },
        "location": {
          "type": "string",
          "description": "Filesystem path or URL for the result file."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "format": {
      "type": "string",
      "description": "Optional format hint: json, yaml, yml, log, or txt."
    },
    "include_raw": {
      "type": "boolean",
      "description": "Include raw text in the response when true.",
      "default": true
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// ResultsLoadParams defines the workflow_results_load tool input.
type ResultsLoadParams struct {
	Source     ListSourceParams `json:"source"`
	FormatHint string           `json:"format,omitempty"`
	IncludeRaw *bool            `json:"include_raw,omitempty"`
}

// ResultsLoadResult is the workflow_results_load tool output payload.
type ResultsLoadResult struct {
	Source        ListSource  `json:"source"`
	Format        string      `json:"format"`
	Payload       interface{} `json:"payload"`
	RawText       string      `json:"raw_text,omitempty"`
	SizeBytes     int64       `json:"size_bytes,omitempty"`
	ContentSHA256 string      `json:"content_sha256,omitempty"`
}

// NewWorkflowResultsLoadTool registers the workflow_results_load tool.
func NewWorkflowResultsLoadTool(
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_results_load",
			Description: "Load workflow result files from disk or URL.",
			InputSchema: json.RawMessage(workflowResultsLoadInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if ctx.Err() != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"context cancelled",
					map[string]string{"error": ctx.Err().Error()},
				)
			}
			payload, err := json.Marshal(arguments)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			var params ResultsLoadParams
			if err := json.Unmarshal(payload, &params); err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			if params.Source.Kind == "" || params.Source.Location == "" {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"source.kind and source.location are required",
					nil,
				)
			}
			includeRaw := true
			if params.IncludeRaw != nil {
				includeRaw = *params.IncludeRaw
			}

			rawText, sizeBytes, shaSum, err := loadResultContent(
				ctx,
				params.Source,
			)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"result load failed",
					map[string]string{"error": err.Error()},
				)
			}

			format, payloadValue, err := parseResultPayload(
				rawText,
				params.FormatHint,
			)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"result parse failed",
					map[string]string{"error": err.Error()},
				)
			}

			result := ResultsLoadResult{
				Source: ListSource{
					Kind:     strings.ToLower(params.Source.Kind),
					Location: params.Source.Location,
				},
				Format:        format,
				Payload:       payloadValue,
				SizeBytes:     sizeBytes,
				ContentSHA256: shaSum,
			}
			if includeRaw {
				result.RawText = rawText
			}

			return renderJSONResult(result, logger)
		},
	}
}

func loadResultContent(
	ctx context.Context,
	source ListSourceParams,
) (string, int64, string, error) {
	kind := strings.ToLower(strings.TrimSpace(source.Kind))
	switch kind {
	case "filesystem":
		path := filepath.Clean(source.Location)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", 0, "", fmt.Errorf("read result file: %w", err)
		}
		sum := sha256.Sum256(data)
		return string(data), int64(len(data)), hex.EncodeToString(sum[:]), nil
	case "url":
		if _, err := url.Parse(source.Location); err != nil {
			return "", 0, "", fmt.Errorf("parse url: %w", err)
		}
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.Location, nil)
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
		return "", 0, "", fmt.Errorf("unsupported source kind %q", source.Kind)
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
