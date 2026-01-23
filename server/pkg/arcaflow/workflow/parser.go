package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// ParsedWorkflow holds the normalized workflow document and schemas.
type ParsedWorkflow struct {
	Workflow         Workflow
	Document         json.RawMessage
	InputSchema      json.RawMessage
	InputSchemaPath  string
	InputExample     json.RawMessage
	OutputSchema     json.RawMessage
	OutputSchemaPath string
}

// Parser extracts schema information from workflow documents.
type Parser struct {
	logger     *slog.Logger
	httpClient *http.Client
}

// ParserOption configures workflow parser behavior.
type ParserOption func(*Parser)

// NewParser constructs a workflow parser with defaults.
func NewParser(options ...ParserOption) *Parser {
	parser := &Parser{
		logger: slog.Default(),
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
	for _, option := range options {
		if option != nil {
			option(parser)
		}
	}
	return parser
}

// WithParserLogger sets the parser logger.
func WithParserLogger(logger *slog.Logger) ParserOption {
	return func(parser *Parser) {
		if logger != nil {
			parser.logger = logger
		}
	}
}

// WithParserHTTPClient sets the HTTP client used for schema references.
func WithParserHTTPClient(client *http.Client) ParserOption {
	return func(parser *Parser) {
		if client != nil {
			parser.httpClient = client
		}
	}
}

// Parse normalizes a workflow document and extracts input/output schemas.
func (parser *Parser) Parse(
	ctx context.Context,
	workflow Workflow,
) (ParsedWorkflow, error) {
	if ctx.Err() != nil {
		return ParsedWorkflow{}, ctx.Err()
	}
	if len(workflow.Content) == 0 {
		return ParsedWorkflow{}, fmt.Errorf("workflow content is empty")
	}

	document, root, err := parseDocument(workflow.Content)
	if err != nil {
		return ParsedWorkflow{}, err
	}

	inputSchema, inputPath, ok, err := inferSchemaFromKey(root, "input")
	if err != nil {
		return ParsedWorkflow{}, err
	}
	if !ok {
		return ParsedWorkflow{}, fmt.Errorf("input schema not found (expected input)")
	}
	inputSchemaIsArcaflow := true

	outputSchema, outputPath, ok, err := extractSchema(
		root,
		[]string{"outputSchema"},
	)
	if err != nil {
		return ParsedWorkflow{}, err
	}
	if !ok {
		if outputs, ok := root["outputs"].(map[string]interface{}); ok {
			outputSchema = buildOutputSchema(outputs)
			outputPath = "outputs"
		} else if outputs, ok := root["output"].(map[string]interface{}); ok {
			outputSchema = buildOutputSchema(outputs)
			outputPath = "output"
		} else {
			outputSchema = nil
			outputPath = ""
		}
	}

	var inputExample json.RawMessage
	if !inputSchemaIsArcaflow {
		inputExample, err = GenerateExampleInput(inputSchema)
		if err != nil {
			return ParsedWorkflow{}, err
		}
	}

	parser.logger.Debug(
		"parsed workflow schemas",
		"workflow_id",
		workflow.ID,
		"input_key",
		inputPath,
		"output_key",
		outputPath,
	)

	return ParsedWorkflow{
		Workflow:         workflow,
		Document:         document,
		InputSchema:      inputSchema,
		InputSchemaPath:  inputPath,
		InputExample:     inputExample,
		OutputSchema:     outputSchema,
		OutputSchemaPath: outputPath,
	}, nil
}

func parseDocument(content []byte) (json.RawMessage, map[string]interface{}, error) {
	root, err := parseContent(content)
	if err != nil {
		return nil, nil, err
	}

	document, err := json.Marshal(root)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal workflow: %w", err)
	}
	return document, root, nil
}

func extractSchema(
	root map[string]interface{},
	keys []string,
) (json.RawMessage, string, bool, error) {
	for _, key := range keys {
		value, ok := root[key]
		if !ok {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, "", false, fmt.Errorf("marshal %s schema: %w", key, err)
		}
		if string(raw) == "null" {
			return nil, "", false, fmt.Errorf("schema %s is null", key)
		}
		return raw, key, true, nil
	}
	return nil, "", false, nil
}

func buildOutputSchema(outputs map[string]interface{}) json.RawMessage {
	properties := make(map[string]interface{}, len(outputs))
	for key := range outputs {
		properties[key] = map[string]interface{}{}
	}
	raw, err := json.Marshal(map[string]interface{}{
		"type":       "object",
		"properties": properties,
	})
	if err != nil {
		return nil
	}
	return raw
}

func inferSchemaFromKey(
	root map[string]interface{},
	key string,
) (json.RawMessage, string, bool, error) {
	value, ok := root[key]
	if !ok {
		return nil, "", false, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, "", false, fmt.Errorf("marshal %s schema: %w", key, err)
	}
	if string(raw) == "null" {
		return nil, "", false, fmt.Errorf("schema %s is null", key)
	}
	return raw, key, true, nil
}
