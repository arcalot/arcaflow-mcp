package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
)

func TestExecutionResourceRead(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	resultPath := filepath.Join(root, "result.json")
	if err := os.WriteFile(resultPath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	provider := NewExecutionResourceProvider(nil)
	uri := "execution://filesystem?location=" + resultPath
	ctx := auth.WithTenantID(context.Background(), "tenant-d")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["format"] != "json" {
		t.Fatalf("expected json format")
	}
}

func TestExecutionLogResourceForcesLog(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	resultPath := filepath.Join(root, "result.json")
	if err := os.WriteFile(resultPath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	provider := NewExecutionResourceProvider(nil)
	uri := "execution-log://filesystem?location=" + resultPath
	ctx := auth.WithTenantID(context.Background(), "tenant-e")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["format"] != "log" {
		t.Fatalf("expected log format")
	}
}

func TestExecutionResourceInvalidIncludeRaw(t *testing.T) {
	t.Parallel()

	provider := NewExecutionResourceProvider(nil)
	uri := "execution://filesystem?location=/tmp/result.json&include_raw=maybe"
	_, handled, errObj := provider.Read(context.Background(), uri)
	if !handled {
		t.Fatalf("expected execution uri to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected include_raw error")
	}
}

func TestExecutionResourceMissingLocation(t *testing.T) {
	t.Parallel()

	provider := NewExecutionResourceProvider(nil)
	uri := "execution://filesystem"
	_, handled, errObj := provider.Read(context.Background(), uri)
	if !handled {
		t.Fatalf("expected execution uri to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected missing location error")
	}
}

func TestExecutionResourceParsesYAML(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	resultPath := filepath.Join(root, "result.yaml")
	if err := os.WriteFile(resultPath, []byte("status: ok\n"), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	provider := NewExecutionResourceProvider(nil)
	uri := "execution://filesystem?location=" + resultPath + "&format=yaml"
	ctx := auth.WithTenantID(context.Background(), "tenant-f")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["format"] != "yaml" {
		t.Fatalf("expected yaml format")
	}
}

func TestExecutionResourceListCaches(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	resultPath := filepath.Join(root, "result.json")
	if err := os.WriteFile(resultPath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	provider := NewExecutionResourceProvider(nil)
	uri := "execution://filesystem?location=" + resultPath
	ctx := auth.WithTenantID(context.Background(), "tenant-g")
	_, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil || !handled {
		t.Fatalf("expected read to succeed")
	}

	items, errObj := provider.List(ctx)
	if errObj != nil {
		t.Fatalf("list resources: %v", errObj)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 cached resource")
	}
}

func TestExecutionResourceParsers(t *testing.T) {
	t.Parallel()

	format, _, err := parseResultPayload(`{"ok":true}`, "")
	if err != nil {
		t.Fatalf("expected json parse: %v", err)
	}
	if format != "json" {
		t.Fatalf("expected json format")
	}

	format, _, err = parseResultPayload("status: ok\n", "")
	if err != nil {
		t.Fatalf("expected yaml parse: %v", err)
	}
	if format != "yaml" {
		t.Fatalf("expected yaml format")
	}
}

func TestExecutionResourceLoadResultContentURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	raw, size, _, err := loadResultContent(
		context.Background(),
		"url",
		server.URL,
	)
	if err != nil {
		t.Fatalf("expected url load: %v", err)
	}
	if size == 0 || raw == "" {
		t.Fatalf("expected payload")
	}
}

func TestExecutionResourceRenderError(t *testing.T) {
	t.Parallel()

	provider := NewExecutionResourceProvider(nil)
	_, errObj := provider.renderResource("execution://x", map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected render error")
	}
}

func TestExecutionResourceLoadResultContentErrors(t *testing.T) {
	t.Parallel()

	if _, _, _, err := loadResultContent(
		context.Background(),
		"unknown",
		"/tmp/result.json",
	); err == nil {
		t.Fatalf("expected unsupported kind error")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	if _, _, _, err := loadResultContent(
		context.Background(),
		"url",
		server.URL,
	); err == nil {
		t.Fatalf("expected url status error")
	}

	if _, _, _, err := loadResultContent(
		context.Background(),
		"url",
		"http://[::1",
	); err == nil {
		t.Fatalf("expected url parse error")
	}
}

func TestExecutionResourceParseResultPayloadErrors(t *testing.T) {
	t.Parallel()

	if _, _, err := parseResultPayload("{bad", "json"); err == nil {
		t.Fatalf("expected json parse error")
	}
	if _, _, err := parseResultPayload("{bad", "yaml"); err == nil {
		t.Fatalf("expected yaml parse error")
	}
	if format, payload, err := parseResultPayload("log-text", "log"); err != nil ||
		format != "log" || payload.(string) != "log-text" {
		t.Fatalf("expected log payload")
	}
	if _, _, err := parseResultPayload("value", "xml"); err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestParseExecutionResourceURIValidation(t *testing.T) {
	t.Parallel()

	if _, handled, err := parseExecutionResourceURI(
		"execution://?location=/tmp",
	); !handled || err == nil {
		t.Fatalf("expected missing kind error")
	}

	if _, handled, err := parseExecutionResourceURI(
		"execution://filesystem",
	); !handled || err == nil {
		t.Fatalf("expected missing location error")
	}

	if _, handled, err := parseExecutionResourceURI(
		"execution://filesystem?location=/tmp&include_raw=maybe",
	); !handled || err == nil {
		t.Fatalf("expected include_raw error")
	}
}
