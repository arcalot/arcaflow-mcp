package plugintools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/pluginmeta"
	"github.com/arcalot/arcaflow-mcp/server/pkg/quay"
)

// testMetaYAML provides a small plugin metadata config
// used across all tests.
const testMetaYAML = `
plugins:
  arcaflow-plugin-fio:
    keywords: ["storage", "benchmark", "fio"]
    category: "storage"
  arcaflow-plugin-stressng:
    keywords: ["stress", "cpu", "memory"]
    category: "stress"
  arcaflow-plugin-iperf3:
    keywords: ["network", "bandwidth"]
    category: "network"
`

// reposJSON builds a Quay repos response with the
// given repositories.
func reposJSON(repos []map[string]string) []byte {
	type repoList struct {
		Repositories []map[string]string `json:"repositories"`
	}
	data, _ := json.Marshal(repoList{
		Repositories: repos,
	})
	return data
}

// tagsJSON builds a Quay tags response.
func tagsJSON(tags []map[string]string) []byte {
	type tagList struct {
		Tags []map[string]string `json:"tags"`
	}
	data, _ := json.Marshal(tagList{Tags: tags})
	return data
}

// setupMockQuay creates an httptest server returning
// known plugin repos and tags. The returned counter
// tracks how many times /api/v1/repository is hit.
func setupMockQuay(
	t *testing.T,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	hits := &atomic.Int32{}
	mux := http.NewServeMux()

	// Repos endpoint — returns 3 plugins for "arcalot"
	mux.HandleFunc(
		"/api/v1/repository",
		func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			ns := r.URL.Query().Get("namespace")
			if ns == "arcalot" {
				_, _ = w.Write(reposJSON([]map[string]string{
					{
						"namespace":   "arcalot",
						"name":        "arcaflow-plugin-fio",
						"description": "FIO benchmark plugin",
					},
					{
						"namespace":   "arcalot",
						"name":        "arcaflow-plugin-stressng",
						"description": "",
					},
					{
						"namespace":   "arcalot",
						"name":        "arcaflow-plugin-iperf3",
						"description": "iperf3 network tests",
					},
				}))
				return
			}
			// Other orgs return empty
			_, _ = w.Write(reposJSON(nil))
		},
	)

	// Tag endpoints per repo
	mux.HandleFunc(
		"/api/v1/repository/arcalot/"+
			"arcaflow-plugin-fio/tag/",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(tagsJSON([]map[string]string{
				{"name": "0.9.0", "manifest_digest": "sha256:aaa"},
				{"name": "1.0.0", "manifest_digest": "sha256:bbb"},
				{"name": "latest", "manifest_digest": "sha256:ccc"},
			}))
		},
	)
	mux.HandleFunc(
		"/api/v1/repository/arcalot/"+
			"arcaflow-plugin-stressng/tag/",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(tagsJSON([]map[string]string{
				{"name": "0.1.0", "manifest_digest": "sha256:ddd"},
			}))
		},
	)
	mux.HandleFunc(
		"/api/v1/repository/arcalot/"+
			"arcaflow-plugin-iperf3/tag/",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(tagsJSON([]map[string]string{
				{"name": "2.3.1", "manifest_digest": "sha256:eee"},
			}))
		},
	)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, hits
}

// newTestService creates a PluginCatalogService wired
// to the mock Quay server and test metadata.
func newTestService(
	t *testing.T,
	srv *httptest.Server,
) *PluginCatalogService {
	t.Helper()
	logger := slog.Default()
	client := quay.NewClient(logger).WithBaseURL(srv.URL)
	meta := pluginmeta.NewCatalogFromBytes(
		[]byte(testMetaYAML),
	)
	svc := NewPluginCatalogService(
		client, meta, logger, 5*time.Minute,
	)
	// Only scan "arcalot" for deterministic tests.
	svc.orgs = []string{"arcalot"}
	return svc
}

func TestPluginListReturnsAllPlugins(t *testing.T) {
	srv, _ := setupMockQuay(t)
	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	res, errObj := tool.Handler(
		context.Background(), map[string]interface{}{},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
	if res.IsError {
		t.Fatal("result marked as error")
	}

	var out PluginListResult
	if err := json.Unmarshal(
		[]byte(res.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if out.Total != 3 {
		t.Errorf("total = %d, want 3", out.Total)
	}
	if out.Filtered != 3 {
		t.Errorf("filtered = %d, want 3", out.Filtered)
	}
	if len(out.Plugins) != 3 {
		t.Fatalf("plugins len = %d, want 3",
			len(out.Plugins))
	}
}

func TestPluginListFilterByCategory(t *testing.T) {
	srv, _ := setupMockQuay(t)
	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	res, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{"category": "storage"},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	var out PluginListResult
	if err := json.Unmarshal(
		[]byte(res.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Total != 3 {
		t.Errorf("total = %d, want 3", out.Total)
	}
	if out.Filtered != 1 {
		t.Errorf("filtered = %d, want 1", out.Filtered)
	}
	if out.Plugins[0].Name != "arcaflow-plugin-fio" {
		t.Errorf(
			"name = %s, want arcaflow-plugin-fio",
			out.Plugins[0].Name,
		)
	}
}

func TestPluginListFilterByCategoryNoMatch(t *testing.T) {
	srv, _ := setupMockQuay(t)
	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	res, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"category": "nonexistent",
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	var out PluginListResult
	if err := json.Unmarshal(
		[]byte(res.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Filtered != 0 {
		t.Errorf("filtered = %d, want 0", out.Filtered)
	}
	if len(out.Plugins) != 0 {
		t.Errorf("plugins len = %d, want 0",
			len(out.Plugins))
	}
}

func TestPluginListFilterByArchitecture(t *testing.T) {
	srv, _ := setupMockQuay(t)
	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	// Default architecture is "unknown", so filtering
	// by "unknown" should return all, and "amd64"
	// should return none.
	tests := []struct {
		arch string
		want int
	}{
		{"unknown", 3},
		{"amd64", 0},
	}

	for _, tc := range tests {
		t.Run(tc.arch, func(t *testing.T) {
			res, errObj := tool.Handler(
				context.Background(),
				map[string]interface{}{
					"architecture": tc.arch,
				},
			)
			if errObj != nil {
				t.Fatalf("error: %s", errObj.Message)
			}
			var out PluginListResult
			if err := json.Unmarshal(
				[]byte(res.Content[0].Text), &out,
			); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.Filtered != tc.want {
				t.Errorf(
					"filtered = %d, want %d",
					out.Filtered, tc.want,
				)
			}
		})
	}
}

func TestPluginListCaching(t *testing.T) {
	srv, hits := setupMockQuay(t)
	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	// First call fetches from Quay.
	_, errObj := tool.Handler(
		context.Background(), map[string]interface{}{},
	)
	if errObj != nil {
		t.Fatalf("call 1 error: %s", errObj.Message)
	}
	firstHits := hits.Load()

	// Second call should use cache.
	_, errObj = tool.Handler(
		context.Background(), map[string]interface{}{},
	)
	if errObj != nil {
		t.Fatalf("call 2 error: %s", errObj.Message)
	}

	if hits.Load() != firstHits {
		t.Errorf(
			"quay hit count changed: %d -> %d",
			firstHits, hits.Load(),
		)
	}
}

func TestPluginListQuayError(t *testing.T) {
	// Server that always returns 500.
	srv := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	t.Cleanup(srv.Close)

	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	res, errObj := tool.Handler(
		context.Background(), map[string]interface{}{},
	)
	// Should not return a tool error — just empty list.
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	var out PluginListResult
	if err := json.Unmarshal(
		[]byte(res.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Total != 0 {
		t.Errorf("total = %d, want 0", out.Total)
	}
}

func TestPluginListQuayErrorWithCache(t *testing.T) {
	srv, _ := setupMockQuay(t)
	svc := newTestService(t, srv)

	// Populate cache.
	plugins, err := svc.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("initial fetch: %v", err)
	}
	if len(plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d",
			len(plugins))
	}

	// Expire the cache manually.
	svc.cacheMu.Lock()
	svc.cacheTime = time.Now().Add(-10 * time.Minute)
	svc.cacheMu.Unlock()

	// Replace server URL with a broken one.
	brokenSrv := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.WriteHeader(
				http.StatusInternalServerError,
			)
		}),
	)
	t.Cleanup(brokenSrv.Close)
	svc.quayClient = quay.NewClient(
		slog.Default(),
	).WithBaseURL(brokenSrv.URL)
	svc.orgs = []string{"arcalot"}

	// Should fall back to stale cache.
	result, err := svc.ListPlugins(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result) != 3 {
		t.Errorf(
			"expected 3 cached plugins, got %d",
			len(result),
		)
	}
}

func TestPluginListEnrichment(t *testing.T) {
	srv, _ := setupMockQuay(t)
	svc := newTestService(t, srv)
	tool := NewPluginListTool(svc, nil)

	res, errObj := tool.Handler(
		context.Background(), map[string]interface{}{},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	var out PluginListResult
	if err := json.Unmarshal(
		[]byte(res.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Find fio plugin and verify enrichment.
	var fio *PluginInfo
	var stressng *PluginInfo
	for i := range out.Plugins {
		switch out.Plugins[i].Name {
		case "arcaflow-plugin-fio":
			fio = &out.Plugins[i]
		case "arcaflow-plugin-stressng":
			stressng = &out.Plugins[i]
		}
	}

	if fio == nil {
		t.Fatal("fio plugin not found")
	}
	if fio.Category != "storage" {
		t.Errorf(
			"fio category = %s, want storage",
			fio.Category,
		)
	}
	if fio.Version != "1.0.0" {
		t.Errorf(
			"fio version = %s, want 1.0.0",
			fio.Version,
		)
	}
	if fio.Description != "FIO benchmark plugin" {
		t.Errorf(
			"fio desc = %s, want FIO benchmark plugin",
			fio.Description,
		)
	}
	if len(fio.Keywords) != 3 {
		t.Errorf(
			"fio keywords len = %d, want 3",
			len(fio.Keywords),
		)
	}
	if fio.Image != "quay.io/arcalot/arcaflow-plugin-fio" {
		t.Errorf("fio image = %s", fio.Image)
	}

	// stressng has no description — should be inferred.
	if stressng == nil {
		t.Fatal("stressng plugin not found")
	}
	want := "Arcaflow plugin: stressng"
	if stressng.Description != want {
		t.Errorf(
			"stressng desc = %q, want %q",
			stressng.Description, want,
		)
	}
	fmt.Printf(
		"enrichment verified: fio=%+v\n", *fio,
	)
}
