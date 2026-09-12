package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/pluginschema"
	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const (
	workflowScheme        = "workflow"
	workflowSchemaScheme  = "workflow-schema"
	workflowExampleScheme = "workflow-example"
	pluginSchemaScheme    = "plugin-schema"
)

// WorkflowResourceProvider exposes workflow schema and example resources.
type WorkflowResourceProvider struct {
	loader *workflow.Loader
	parser *workflow.Parser
	logger *slog.Logger
	mu     sync.RWMutex
	cache  map[string]map[string]protocol.ResourceItem
}

// NewWorkflowResourceProvider constructs a workflow resource provider.
func NewWorkflowResourceProvider(
	loader *workflow.Loader,
	parser *workflow.Parser,
	logger *slog.Logger,
) *WorkflowResourceProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkflowResourceProvider{
		loader: loader,
		parser: parser,
		logger: logger,
		cache:  make(map[string]map[string]protocol.ResourceItem),
	}
}

// List returns cached resources for the current tenant.
func (provider *WorkflowResourceProvider) List(
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

// Read resolves workflow schema or example resources from URI metadata.
func (provider *WorkflowResourceProvider) Read(
	ctx context.Context,
	uri string,
) (*protocol.ResourceContent, bool, *protocol.ErrorObject) {
	kind, params, ok, err := parseWorkflowResourceURI(uri)
	if !ok {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			"invalid workflow resource uri",
			map[string]string{"error": err.Error()},
		)
	}
	if provider.loader == nil || provider.parser == nil {
		return nil, true, resourceError(
			protocol.ErrInternal,
			"workflow resources not configured",
			nil,
		)
	}

	index, err := loadIndex(ctx, provider.loader, params.Source)
	if err != nil {
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			fmt.Sprintf("workflow source load failed: %s", err.Error()),
			map[string]string{"error": err.Error()},
		)
	}

	selected, err := selectWorkflow(index.Workflows, params.Selector)
	if err != nil {
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			"workflow selection failed",
			map[string]string{"error": err.Error()},
		)
	}

	switch kind {
	case workflowScheme:
		payload := provider.buildWorkflowPayload(selected)
		content, errObj := provider.renderResource(uri, payload)
		if errObj != nil {
			return nil, true, errObj
		}
		provider.cacheResource(ctx, uri, selected, "Workflow definition")
		return content, true, nil
	case workflowSchemaScheme:
		payload, err := provider.buildSchemaPayload(ctx, selected)
		if err != nil {
			return nil, true, err
		}
		content, errObj := provider.renderResource(uri, payload)
		if errObj != nil {
			return nil, true, errObj
		}
		provider.cacheResource(ctx, uri, selected, "Resolved workflow schema")
		return content, true, nil
	case workflowExampleScheme:
		payload, err := provider.buildExamplePayload(ctx, selected)
		if err != nil {
			return nil, true, err
		}
		content, errObj := provider.renderResource(uri, payload)
		if errObj != nil {
			return nil, true, errObj
		}
		provider.cacheResource(ctx, uri, selected, "Example workflow input")
		return content, true, nil
	case pluginSchemaScheme:
		payload, err := provider.buildPluginSchemaPayload(
			ctx,
			selected,
			params.StepID,
		)
		if err != nil {
			return nil, true, err
		}
		content, errObj := provider.renderResource(uri, payload)
		if errObj != nil {
			return nil, true, errObj
		}
		provider.cacheResource(ctx, uri, selected, "Plugin schema references")
		return content, true, nil
	default:
		return nil, true, resourceError(
			protocol.ErrInvalidParams,
			"unsupported workflow resource",
			nil,
		)
	}
}

type workflowResourceParams struct {
	Source   sourceParams
	Selector selectorParams
	StepID   string
}

type sourceParams struct {
	Kind     string
	Location string
	Ref      string
	Subdir   string
}

type selectorParams struct {
	ID   string
	Path string
}

func parseWorkflowResourceURI(
	rawURI string,
) (string, workflowResourceParams, bool, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", workflowResourceParams{}, false, err
	}
	switch parsed.Scheme {
	case workflowScheme, workflowSchemaScheme, workflowExampleScheme,
		pluginSchemaScheme:
	default:
		return "", workflowResourceParams{}, false, nil
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
	if kind == "" {
		return "", workflowResourceParams{}, true, fmt.Errorf(
			"source kind required",
		)
	}
	if location == "" {
		return "", workflowResourceParams{}, true, fmt.Errorf(
			"source location required",
		)
	}
	selectorPath := strings.TrimSpace(query.Get("path"))
	if selectorPath == "" && strings.TrimSpace(parsed.Path) != "" {
		selectorPath = strings.TrimPrefix(parsed.Path, "/")
	}
	params := workflowResourceParams{
		Source: sourceParams{
			Kind:     kind,
			Location: location,
			Ref:      strings.TrimSpace(query.Get("ref")),
			Subdir:   strings.TrimSpace(query.Get("subdir")),
		},
		Selector: selectorParams{
			ID:   strings.TrimSpace(query.Get("id")),
			Path: selectorPath,
		},
		StepID: strings.TrimSpace(query.Get("step_id")),
	}
	return parsed.Scheme, params, true, nil
}

type workflowResourceMetadata struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Path   string                 `json:"path"`
	Source workflowResourceSource `json:"source"`
}

type workflowResourceSource struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
	Ref      string `json:"ref,omitempty"`
	Subdir   string `json:"subdir,omitempty"`
}

type schemaKeyMetadata struct {
	InputKey  string `json:"input_key,omitempty"`
	OutputKey string `json:"output_key,omitempty"`
}

type workflowSchemaResource struct {
	Workflow         workflowResourceMetadata `json:"workflow"`
	InputJSONSchema  json.RawMessage          `json:"input_json_schema"`
	OutputJSONSchema json.RawMessage          `json:"output_json_schema,omitempty"`
	ExampleInput     json.RawMessage          `json:"example_input,omitempty"`
	ExampleGenerated bool                     `json:"example_generated,omitempty"`
	SchemaKeys       schemaKeyMetadata        `json:"schema_keys"`
}

type workflowExampleResource struct {
	Workflow     workflowResourceMetadata `json:"workflow"`
	ExampleInput json.RawMessage          `json:"example_input"`
	Generated    bool                     `json:"generated"`
	InputKey     string                   `json:"input_key,omitempty"`
}

type workflowDefinitionResource struct {
	Workflow workflowResourceMetadata `json:"workflow"`
	Content  string                   `json:"content"`
}

type pluginSchemaEntry struct {
	StepID   string          `json:"step_id"`
	Location string          `json:"location"`
	Schema   json.RawMessage `json:"schema"`
}

type pluginSchemaResource struct {
	Workflow workflowResourceMetadata `json:"workflow"`
	Schemas  []pluginSchemaEntry      `json:"schemas"`
}

func (provider *WorkflowResourceProvider) buildWorkflowPayload(
	selected workflow.Workflow,
) workflowDefinitionResource {
	return workflowDefinitionResource{
		Workflow: buildWorkflowMetadata(selected),
		Content:  string(selected.Content),
	}
}

func (provider *WorkflowResourceProvider) buildSchemaPayload(
	ctx context.Context,
	selected workflow.Workflow,
) (workflowSchemaResource, *protocol.ErrorObject) {
	parsed, err := provider.parser.Parse(ctx, selected)
	if err != nil {
		return workflowSchemaResource{}, resourceError(
			protocol.ErrInvalidParams,
			"workflow schema parse failed",
			map[string]string{"error": err.Error()},
		)
	}

	resolver := workflow.NewInputSchemaResolver()
	resolvedInput, err := resolver.ResolveInputJSONSchema(ctx, selected)
	if err != nil {
		details := map[string]string{"error": err.Error()}
		var resolutionErr workflow.NamespaceResolutionError
		if errors.As(err, &resolutionErr) && resolutionErr.Hint != "" {
			details["hint"] = resolutionErr.Hint
		}
		if strings.Contains(err.Error(), "no container runtime available") {
			details["hint"] = "Install podman or docker to resolve plugin schemas."
		}
		return workflowSchemaResource{}, resourceError(
			protocol.ErrInvalidParams,
			"workflow input schema resolution failed",
			details,
		)
	}

	example := parsed.InputExample
	generated := false
	if len(example) == 0 {
		example, err = workflow.GenerateExampleInput(resolvedInput)
		if err != nil {
			return workflowSchemaResource{}, resourceError(
				protocol.ErrInvalidParams,
				"example input generation failed",
				map[string]string{"error": err.Error()},
			)
		}
		generated = true
	}

	payload := workflowSchemaResource{
		Workflow:         buildWorkflowMetadata(selected),
		InputJSONSchema:  resolvedInput,
		OutputJSONSchema: parsed.OutputSchema,
		ExampleInput:     example,
		ExampleGenerated: generated,
		SchemaKeys: schemaKeyMetadata{
			InputKey:  parsed.InputSchemaPath,
			OutputKey: parsed.OutputSchemaPath,
		},
	}
	return payload, nil
}

func (provider *WorkflowResourceProvider) buildExamplePayload(
	ctx context.Context,
	selected workflow.Workflow,
) (workflowExampleResource, *protocol.ErrorObject) {
	parsed, err := provider.parser.Parse(ctx, selected)
	if err != nil {
		return workflowExampleResource{}, resourceError(
			protocol.ErrInvalidParams,
			"workflow schema parse failed",
			map[string]string{"error": err.Error()},
		)
	}

	example := parsed.InputExample
	generated := false
	if len(example) == 0 {
		resolver := workflow.NewInputSchemaResolver()
		resolvedInput, err := resolver.ResolveInputJSONSchema(ctx, selected)
		if err != nil {
			details := map[string]string{"error": err.Error()}
			var resolutionErr workflow.NamespaceResolutionError
			if errors.As(err, &resolutionErr) && resolutionErr.Hint != "" {
				details["hint"] = resolutionErr.Hint
			}
			if strings.Contains(err.Error(), "no container runtime available") {
				details["hint"] = "Install podman or docker to resolve plugin schemas."
			}
			return workflowExampleResource{}, resourceError(
				protocol.ErrInvalidParams,
				"workflow input schema resolution failed",
				details,
			)
		}
		example, err = workflow.GenerateExampleInput(resolvedInput)
		if err != nil {
			return workflowExampleResource{}, resourceError(
				protocol.ErrInvalidParams,
				"example input generation failed",
				map[string]string{"error": err.Error()},
			)
		}
		generated = true
	}

	payload := workflowExampleResource{
		Workflow:     buildWorkflowMetadata(selected),
		ExampleInput: example,
		Generated:    generated,
		InputKey:     parsed.InputSchemaPath,
	}
	return payload, nil
}

func (provider *WorkflowResourceProvider) buildPluginSchemaPayload(
	ctx context.Context,
	selected workflow.Workflow,
	stepID string,
) (pluginSchemaResource, *protocol.ErrorObject) {
	handler := pluginschema.NewHandler(provider.logger)
	entries, err := handler.LoadFromWorkflow(
		ctx,
		selected.Content,
		selected.LocalPath,
	)
	if err != nil {
		return pluginSchemaResource{}, resourceError(
			protocol.ErrInvalidParams,
			"plugin schema load failed",
			map[string]string{"error": err.Error()},
		)
	}

	schemas := make([]pluginSchemaEntry, 0, len(entries))
	for _, entry := range entries {
		if stepID != "" && entry.StepID != stepID {
			continue
		}
		schemas = append(schemas, pluginSchemaEntry{
			StepID:   entry.StepID,
			Location: entry.Location,
			Schema:   entry.Schema,
		})
	}
	if stepID != "" && len(schemas) == 0 {
		details := map[string]string{"step_id": stepID}
		if len(entries) == 0 {
			details["hint"] = "No plugin schema references found in workflow."
		}
		return pluginSchemaResource{}, resourceError(
			protocol.ErrInvalidParams,
			"no plugin schemas found for step",
			details,
		)
	}

	payload := pluginSchemaResource{
		Workflow: buildWorkflowMetadata(selected),
		Schemas:  schemas,
	}
	return payload, nil
}

func (provider *WorkflowResourceProvider) renderResource(
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

func (provider *WorkflowResourceProvider) cacheResource(
	ctx context.Context,
	uri string,
	selected workflow.Workflow,
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
		Name:        selected.Name,
		Description: description,
		MimeType:    "application/json",
	}
}

func buildWorkflowMetadata(selected workflow.Workflow) workflowResourceMetadata {
	return workflowResourceMetadata{
		ID:   selected.ID,
		Name: selected.Name,
		Path: selected.Path,
		Source: workflowResourceSource{
			Kind:     string(selected.Source.Kind),
			Location: selected.Source.Location,
			Ref:      selected.Source.Ref,
			Subdir:   selected.Source.Subdir,
		},
	}
}

func tenantFromContext(ctx context.Context) string {
	tenantID, ok := auth.TenantIDFromContext(ctx)
	if !ok || tenantID == "" {
		return "local"
	}
	return tenantID
}

func loadIndex(
	ctx context.Context,
	loader *workflow.Loader,
	source sourceParams,
) (workflow.WorkflowIndex, error) {
	kind := strings.ToLower(strings.TrimSpace(source.Kind))
	switch kind {
	case string(workflow.SourceFilesystem):
		return loader.LoadFromFilesystem(ctx, source.Location)
	case string(workflow.SourceURL):
		return loader.LoadFromURL(ctx, source.Location)
	case string(workflow.SourceGit):
		return loader.LoadFromGit(ctx, source.Location, source.Ref, source.Subdir)
	default:
		return workflow.WorkflowIndex{}, fmt.Errorf(
			"unsupported source kind %q",
			source.Kind,
		)
	}
}

func selectWorkflow(
	workflows []workflow.Workflow,
	selector selectorParams,
) (workflow.Workflow, error) {
	if len(workflows) == 0 {
		return workflow.Workflow{}, fmt.Errorf("no workflows found")
	}
	selector.ID = strings.TrimSpace(selector.ID)
	selector.Path = strings.TrimSpace(selector.Path)
	if selector.ID == "" && selector.Path == "" {
		if len(workflows) == 1 {
			return workflows[0], nil
		}
		return workflow.Workflow{}, fmt.Errorf("workflow selector required")
	}
	if selector.ID != "" {
		for _, item := range workflows {
			if item.ID == selector.ID {
				return item, nil
			}
		}
		return workflow.Workflow{}, fmt.Errorf("workflow id not found")
	}
	for _, item := range workflows {
		if item.Path == selector.Path || item.Name == selector.Path {
			return item, nil
		}
	}
	return workflow.Workflow{}, fmt.Errorf("workflow path not found")
}

func resourceError(
	code int,
	message string,
	data interface{},
) *protocol.ErrorObject {
	return &protocol.ErrorObject{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
