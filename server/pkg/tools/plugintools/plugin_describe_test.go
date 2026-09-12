package plugintools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
)

// mockSchemaProvider implements SchemaProvider for tests.
type mockSchemaProvider struct {
	hasRuntime bool
	schema     []byte
	err        error
	callCount  int
}

func (m *mockSchemaProvider) HasRuntime() bool {
	return m.hasRuntime
}

func (m *mockSchemaProvider) FullSchema(
	_ context.Context,
	_ string,
) ([]byte, error) {
	m.callCount++
	return m.schema, m.err
}

// testSchemaYAML is a minimal valid Arcaflow plugin
// schema in the Go SDK output format.
const testSchemaYAML = `steps:
  workload:
    id: workload
    input:
      root: workload_input
      objects:
        workload_input:
          id: workload_input
          properties:
            threads:
              type:
                type_id: integer
              required: true
    outputs:
      success:
        schema:
          root: workload_output
          objects:
            workload_output:
              id: workload_output
              properties:
                throughput:
                  type:
                    type_id: float
    display:
      name: "Run workload"
      description: "Execute the benchmark workload"
`

// testSchemaWrapped simulates Go SDK output which wraps
// the schema under a "serialized_schema" key.
var testSchemaWrapped = []byte(
	"serialized_schema:\n" +
		"  steps:\n" +
		"    workload:\n" +
		"      id: workload\n" +
		"      input:\n" +
		"        root: workload_input\n" +
		"      outputs:\n" +
		"        success:\n" +
		"          schema:\n" +
		"            root: workload_output\n" +
		"      display:\n" +
		"        name: Run workload\n" +
		"        description: Execute the benchmark\n",
)

func TestResolveImageRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "bare plugin name",
			input: "arcaflow-plugin-fio",
			want:  "quay.io/arcalot/arcaflow-plugin-fio",
		},
		{
			name:  "full arcalot ref",
			input: "quay.io/arcalot/arcaflow-plugin-fio",
			want:  "quay.io/arcalot/arcaflow-plugin-fio",
		},
		{
			name: "redhat-performance ref",
			input: "quay.io/redhat-performance/" +
				"arcaflow-plugin-fio",
			want: "quay.io/redhat-performance/" +
				"arcaflow-plugin-fio",
		},
		{
			name:    "rejected evil registry",
			input:   "evil.io/malware",
			wantErr: true,
		},
		{
			name:    "similar org rejected",
			input:   "quay.io/arcalot-evil/malware",
			wantErr: true,
		},
		{
			name:  "strips tag from full ref",
			input: "quay.io/arcalot/arcaflow-plugin-fio:v1",
			want:  "quay.io/arcalot/arcaflow-plugin-fio",
		},
		{
			name:  "strips tag from bare name",
			input: "arcaflow-plugin-fio:0.9.0",
			want:  "quay.io/arcalot/arcaflow-plugin-fio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveImageRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"expected error for %q",
						tt.input,
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf(
					"got %q, want %q", got, tt.want,
				)
			}
		})
	}
}

func TestPluginDescribeNoRuntime(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{hasRuntime: false}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin": "arcaflow-plugin-fio",
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}
	var desc PluginDescribeResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &desc,
	); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if desc.SchemasAvailable {
		t.Error("expected schemas_available=false")
	}
	if desc.Name != "arcaflow-plugin-fio" {
		t.Errorf("got name %q", desc.Name)
	}
}

func TestPluginDescribeWithSchema(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{
		hasRuntime: true,
		schema:     []byte(testSchemaYAML),
	}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin":  "arcaflow-plugin-fio",
			"version": "0.9.0",
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}
	var desc PluginDescribeResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &desc,
	); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !desc.SchemasAvailable {
		t.Error("expected schemas_available=true")
	}
	if len(desc.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(desc.Steps))
	}
	step, ok := desc.Steps["workload"]
	if !ok {
		t.Fatal("missing workload step")
	}
	if step.ID != "workload" {
		t.Errorf("step ID = %q", step.ID)
	}
	if step.Display == nil {
		t.Fatal("expected display metadata")
	}
	if step.Display.Name != "Run workload" {
		t.Errorf("display name = %q", step.Display.Name)
	}
	if step.InputSchema == nil {
		t.Error("expected input_schema")
	}
	if step.Outputs == nil {
		t.Error("expected outputs")
	}
	if desc.DefaultStep != "workload" {
		t.Errorf(
			"default_step = %q, want workload",
			desc.DefaultStep,
		)
	}
	// Description should come from step display.
	if desc.Description !=
		"Execute the benchmark workload" {
		t.Errorf(
			"description = %q", desc.Description,
		)
	}
}

func TestPluginDescribeWrappedSchema(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{
		hasRuntime: true,
		schema:     testSchemaWrapped,
	}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin":  "arcaflow-plugin-fio",
			"version": "1.0.0",
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}
	var desc PluginDescribeResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &desc,
	); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !desc.SchemasAvailable {
		t.Error("expected schemas_available=true")
	}
	if _, ok := desc.Steps["workload"]; !ok {
		t.Error("missing workload step in wrapped schema")
	}
}

func TestPluginDescribeSchemaError(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{
		hasRuntime: true,
		err:        fmt.Errorf("container failed"),
	}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin": "arcaflow-plugin-fio",
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected tool error: %v", errObj)
	}
	// Should fall back gracefully with
	// schemas_available=false.
	var desc PluginDescribeResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &desc,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if desc.SchemasAvailable {
		t.Error("expected schemas_available=false")
	}
}

func TestPluginDescribeInvalidYAML(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{
		hasRuntime: true,
		schema:     []byte("{{not yaml"),
	}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin": "arcaflow-plugin-fio",
		},
	)
	if errObj == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestPluginDescribeCaching(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{
		hasRuntime: true,
		schema:     []byte(testSchemaYAML),
	}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	args := map[string]interface{}{
		"plugin":  "arcaflow-plugin-fio",
		"version": "0.9.0",
	}

	// First call — hits provider.
	_, errObj := tool.Handler(
		context.Background(), args,
	)
	if errObj != nil {
		t.Fatalf("first call error: %v", errObj)
	}
	if provider.callCount != 1 {
		t.Fatalf(
			"expected 1 call, got %d",
			provider.callCount,
		)
	}

	// Second call — should use cache.
	_, errObj = tool.Handler(
		context.Background(), args,
	)
	if errObj != nil {
		t.Fatalf("second call error: %v", errObj)
	}
	if provider.callCount != 1 {
		t.Fatalf(
			"expected 1 call (cached), got %d",
			provider.callCount,
		)
	}
}

func TestPluginDescribeLatestNotCached(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{
		hasRuntime: true,
		schema:     []byte(testSchemaYAML),
	}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	args := map[string]interface{}{
		"plugin": "arcaflow-plugin-fio",
		// version omitted → defaults to "latest"
	}

	_, _ = tool.Handler(context.Background(), args)
	_, _ = tool.Handler(context.Background(), args)

	// "latest" should NOT be cached — provider called
	// both times.
	if provider.callCount != 2 {
		t.Fatalf(
			"expected 2 calls for latest, got %d",
			provider.callCount,
		)
	}
}

func TestPluginDescribeMissingPlugin(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{hasRuntime: true}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{},
	)
	if errObj == nil {
		t.Fatal("expected error for missing plugin")
	}
}

func TestPluginDescribeRejectedImage(t *testing.T) {
	t.Parallel()
	provider := &mockSchemaProvider{hasRuntime: true}
	tool := NewPluginDescribeTool(
		provider, slog.Default(),
	)
	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin": "evil.io/malware",
		},
	)
	if errObj == nil {
		t.Fatal("expected error for rejected image")
	}
}

func TestPluginDescribeNilProvider(t *testing.T) {
	t.Parallel()
	tool := NewPluginDescribeTool(nil, slog.Default())
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"plugin": "arcaflow-plugin-fio",
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}
	var desc PluginDescribeResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &desc,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if desc.SchemasAvailable {
		t.Error("expected schemas_available=false")
	}
}

func TestParseSchemaOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		wantKey string
		wantErr bool
	}{
		{
			name:    "direct schema",
			input:   []byte(testSchemaYAML),
			wantKey: "steps",
		},
		{
			name:    "wrapped schema",
			input:   testSchemaWrapped,
			wantKey: "steps",
		},
		{
			name:    "invalid YAML",
			input:   []byte("{{not yaml"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			schema, err := parseSchemaOutput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := schema[tt.wantKey]; !ok {
				t.Errorf(
					"missing key %q in schema",
					tt.wantKey,
				)
			}
		})
	}
}

func TestExtractSteps(t *testing.T) {
	t.Parallel()
	schema, err := parseSchemaOutput(
		[]byte(testSchemaYAML),
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	steps := extractSteps(schema)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	step, ok := steps["workload"]
	if !ok {
		t.Fatal("missing workload step")
	}
	if step.ID != "workload" {
		t.Errorf("step ID = %q", step.ID)
	}
	if step.Display == nil {
		t.Fatal("expected display")
	}
	if step.Display.Name != "Run workload" {
		t.Errorf("display.name = %q", step.Display.Name)
	}
}
