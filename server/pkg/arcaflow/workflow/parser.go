package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	inputSchema, inputPath, ok, err := parser.extractSchemaWithRefs(
		ctx,
		workflow,
		root,
		inputSchemaKeys(),
		inputSchemaRefKeys(),
		"input",
	)
	if err != nil {
		return ParsedWorkflow{}, err
	}
	inputSchemaIsArcaflow := false
	if !ok {
		inputSchema, inputPath, ok, err = inferSchemaFromKey(root, "input")
		if err != nil {
			return ParsedWorkflow{}, err
		}
		if ok {
			inputSchemaIsArcaflow = true
		}
	}
	if !ok {
		return ParsedWorkflow{}, fmt.Errorf(
			"input schema not found (expected %s, input.schema, or input)",
			formatKeys(inputSchemaKeys()),
		)
	}

	outputSchema, outputPath, ok, err := parser.extractSchemaWithRefs(
		ctx,
		workflow,
		root,
		outputSchemaKeys(),
		outputSchemaRefKeys(),
		"output",
	)
	if err != nil {
		return ParsedWorkflow{}, err
	}
	if !ok {
		outputSchema, outputPath, ok, err = inferSchemaFromKey(root, "outputs")
		if err != nil {
			return ParsedWorkflow{}, err
		}
	}
	if !ok {
		outputSchema, outputPath, ok, err = inferSchemaFromKey(root, "output")
		if err != nil {
			return ParsedWorkflow{}, err
		}
	}
	if !ok {
		return ParsedWorkflow{}, fmt.Errorf(
			"output schema not found (expected %s, output.schema, outputs, or output)",
			formatKeys(outputSchemaKeys()),
		)
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
		"input_schema_path",
		inputPath,
		"output_schema_path",
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

func inputSchemaKeys() []string {
	return []string{"input_schema", "inputSchema"}
}

func outputSchemaKeys() []string {
	return []string{"output_schema", "outputSchema"}
}

func inputSchemaRefKeys() []string {
	return []string{"input_schema_ref", "inputSchemaRef"}
}

func outputSchemaRefKeys() []string {
	return []string{"output_schema_ref", "outputSchemaRef"}
}

func (parser *Parser) extractSchemaWithRefs(
	ctx context.Context,
	workflow Workflow,
	root map[string]interface{},
	keys []string,
	refKeys []string,
	nestedKey string,
) (json.RawMessage, string, bool, error) {
	ref, refPath, ok, err := extractSchemaRef(root, refKeys)
	if err != nil {
		return nil, "", false, err
	}
	if ok {
		raw, err := parser.resolveSchemaRef(ctx, workflow, ref)
		if err != nil {
			return nil, "", false, err
		}
		return raw, refPath, true, nil
	}

	raw, path, ok, err := extractSchema(root, keys)
	if err != nil || ok {
		return raw, path, ok, err
	}

	raw, path, ok, err = extractNestedSchema(root, nestedKey)
	if err != nil {
		return nil, "", false, err
	}
	return raw, path, ok, nil
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

func extractNestedSchema(
	root map[string]interface{},
	key string,
) (json.RawMessage, string, bool, error) {
	value, ok := root[key]
	if !ok {
		return nil, "", false, nil
	}
	nested, ok := value.(map[string]interface{})
	if !ok {
		return nil, "", false, fmt.Errorf("%s section is not an object", key)
	}
	schema, ok := nested["schema"]
	if !ok {
		return nil, "", false, nil
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, "", false, fmt.Errorf("marshal %s.schema: %w", key, err)
	}
	if string(raw) == "null" {
		return nil, "", false, fmt.Errorf("schema %s.schema is null", key)
	}
	return raw, fmt.Sprintf("%s.schema", key), true, nil
}

func extractSchemaRef(
	root map[string]interface{},
	keys []string,
) (string, string, bool, error) {
	for _, key := range keys {
		value, ok := root[key]
		if !ok {
			continue
		}
		ref, ok := value.(string)
		if !ok {
			return "", "", false, fmt.Errorf("%s must be a string", key)
		}
		if strings.TrimSpace(ref) == "" {
			return "", "", false, fmt.Errorf("%s cannot be empty", key)
		}
		return ref, key, true, nil
	}
	return "", "", false, nil
}

func formatKeys(keys []string) string {
	return fmt.Sprintf("%q", keys)
}

func (parser *Parser) resolveSchemaRef(
	ctx context.Context,
	workflow Workflow,
	ref string,
) (json.RawMessage, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if isHTTPURL(ref) {
		return parser.loadSchemaFromURL(ctx, ref)
	}

	path := ref
	if !filepath.IsAbs(path) {
		if workflow.LocalPath == "" {
			return nil, fmt.Errorf("workflow local path missing for schema ref %s", ref)
		}
		path = filepath.Join(filepath.Dir(workflow.LocalPath), ref)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema ref %s: %w", ref, err)
	}
	return parseSchemaContent(data)
}

func (parser *Parser) loadSchemaFromURL(
	ctx context.Context,
	location string,
) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, fmt.Errorf("build schema ref request: %w", err)
	}
	resp, err := parser.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch schema ref: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			parser.logger.Warn(
				"close schema ref response",
				"error",
				err,
			)
		}
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("schema ref status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read schema ref body: %w", err)
	}
	return parseSchemaContent(data)
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
