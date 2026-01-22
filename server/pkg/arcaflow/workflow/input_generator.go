package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ExportFormat defines the output format for generated input files.
type ExportFormat string

const (
	// ExportFormatJSON emits JSON payloads.
	ExportFormatJSON ExportFormat = "json"
	// ExportFormatYAML emits YAML payloads.
	ExportFormatYAML ExportFormat = "yaml"
)

// ExportMetadata includes validation confirmation for generated inputs.
type ExportMetadata struct {
	WorkflowID      string
	InputSchemaPath string
	Validated       bool
	ValidatedAt     time.Time
}

// ExportedInput represents a validated input payload and associated metadata.
type ExportedInput struct {
	Format   ExportFormat
	Payload  []byte
	Metadata ExportMetadata
}

// GenerateInputFile validates and exports a workflow input payload.
func GenerateInputFile(
	ctx context.Context,
	parsed ParsedWorkflow,
	inputPayload []byte,
	format ExportFormat,
) (ExportedInput, error) {
	if ctx.Err() != nil {
		return ExportedInput{}, ctx.Err()
	}
	if format == "" {
		format = ExportFormatJSON
	}
	validator := NewInputValidator()
	result, err := validator.Validate(ctx, parsed.Workflow, inputPayload)
	if err != nil {
		return ExportedInput{}, err
	}
	if !result.Valid {
		return ExportedInput{}, &ValidationError{Issues: result.Issues}
	}

	root, err := parseAny(result.NormalizedInput)
	if err != nil {
		return ExportedInput{}, err
	}

	var payload []byte
	switch format {
	case ExportFormatJSON:
		payload, err = marshalDeterministicJSON(root)
	case ExportFormatYAML:
		payload, err = marshalDeterministicYAML(root)
	default:
		return ExportedInput{}, fmt.Errorf("unsupported export format %q", format)
	}
	if err != nil {
		return ExportedInput{}, err
	}

	return ExportedInput{
		Format:  format,
		Payload: payload,
		Metadata: ExportMetadata{
			WorkflowID:      parsed.Workflow.ID,
			InputSchemaPath: parsed.InputSchemaPath,
			Validated:       true,
			ValidatedAt:     time.Now().UTC(),
		},
	}, nil
}

// ValidationError wraps workflow input validation issues.
type ValidationError struct {
	Issues []ValidationIssue
}

func (err *ValidationError) Error() string {
	if len(err.Issues) == 0 {
		return "workflow input validation failed"
	}
	parts := make([]string, 0, len(err.Issues))
	for _, issue := range err.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Path, issue.Message))
	}
	return "workflow input validation failed: " + strings.Join(parts, "; ")
}

func marshalDeterministicJSON(value interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	if err := writeJSON(buffer, value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeJSON(buffer *bytes.Buffer, value interface{}) error {
	switch typed := value.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		buffer.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buffer.WriteByte(',')
			}
			keyBytes, _ := json.Marshal(key)
			buffer.Write(keyBytes)
			buffer.WriteByte(':')
			if err := writeJSON(buffer, typed[key]); err != nil {
				return err
			}
		}
		buffer.WriteByte('}')
		return nil
	case []interface{}:
		buffer.WriteByte('[')
		for i, entry := range typed {
			if i > 0 {
				buffer.WriteByte(',')
			}
			if err := writeJSON(buffer, entry); err != nil {
				return err
			}
		}
		buffer.WriteByte(']')
		return nil
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return fmt.Errorf("marshal json value: %w", err)
		}
		buffer.Write(raw)
		return nil
	}
}

func marshalDeterministicYAML(value interface{}) ([]byte, error) {
	node := buildYAMLNode(value)
	buffer := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		_ = encoder.Close()
		return nil, fmt.Errorf("encode yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close yaml encoder: %w", err)
	}
	return buffer.Bytes(), nil
}

func buildYAMLNode(value interface{}) *yaml.Node {
	switch typed := value.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		node := &yaml.Node{Kind: yaml.MappingNode}
		for _, key := range keys {
			node.Content = append(node.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: key,
				Tag:   "!!str",
			}, buildYAMLNode(typed[key]))
		}
		return node
	case []interface{}:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for _, entry := range typed {
			node.Content = append(node.Content, buildYAMLNode(entry))
		}
		return node
	default:
		node := &yaml.Node{Kind: yaml.ScalarNode}
		switch scalar := typed.(type) {
		case nil:
			node.Tag = "!!null"
			node.Value = "null"
		case bool:
			node.Tag = "!!bool"
			if scalar {
				node.Value = "true"
			} else {
				node.Value = "false"
			}
		case float64:
			node.Tag = "!!float"
			node.Value = strconv.FormatFloat(scalar, 'f', -1, 64)
		case string:
			node.Tag = "!!str"
			node.Value = scalar
		default:
			node.Tag = "!!str"
			node.Value = fmt.Sprintf("%v", scalar)
		}
		return node
	}
}
