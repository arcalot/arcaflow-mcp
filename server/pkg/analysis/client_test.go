package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyzeSuccess(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/analysis/summary" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		response := map[string]interface{}{
			"analysis": map[string]interface{}{
				"success_rate": 1.0,
				"metric_stats": map[string]interface{}{},
				"record_count": 1,
				"findings":     []interface{}{},
			},
			"suggestions": []interface{}{},
		}
		_ = json.NewEncoder(writer).Encode(response)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("socket operations not permitted: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Analyze(context.Background(), AnalyzeRequest{
		Results: []ResultPayload{
			{Format: "json", Payload: json.RawMessage(`{"ok":true}`)},
		},
	})
	if err != nil {
		t.Fatalf("analyze request failed: %v", err)
	}
	if resp.Analysis.SuccessRate == nil || *resp.Analysis.SuccessRate != 1.0 {
		t.Fatalf("expected success rate 1.0")
	}
	if resp.Analysis.RecordCount != 1 {
		t.Fatalf("expected record count 1")
	}
}

func TestAnalyzeStatusError(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("socket operations not permitted: %v", err)
	}
	server := httptest.NewUnstartedServer(http.NotFoundHandler())
	server.Listener = listener
	server.Start()
	defer server.Close()

	client := NewClient(server.URL)
	_, err = client.Analyze(context.Background(), AnalyzeRequest{})
	if err == nil {
		t.Fatalf("expected error on non-2xx response")
	}
}

func TestAnalyzeMissingBaseURL(t *testing.T) {
	t.Parallel()

	client := NewClient("")
	_, err := client.Analyze(context.Background(), AnalyzeRequest{})
	if err == nil {
		t.Fatalf("expected error for missing base url")
	}
}

func TestAnalyzeCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("http://example.com")
	_, err := client.Analyze(ctx, AnalyzeRequest{})
	if err == nil {
		t.Fatalf("expected error for canceled context")
	}
}

func TestAnalyzeDecodeError(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("socket operations not permitted: %v", err)
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("{invalid-json"))
	})
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	defer server.Close()

	client := NewClient(server.URL)
	_, err = client.Analyze(context.Background(), AnalyzeRequest{})
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestAnalyzeRequestBuildError(t *testing.T) {
	t.Parallel()
	client := NewClient("http://[::1")
	_, err := client.Analyze(context.Background(), AnalyzeRequest{})
	if err == nil {
		t.Fatalf("expected request build error")
	}
}

type errorRoundTripper struct{}

func (errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errRoundTrip
}

var errRoundTrip = errors.New("round trip failed")

func TestAnalyzeSendError(t *testing.T) {
	t.Parallel()
	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: errorRoundTripper{},
	}
	_, err := client.Analyze(context.Background(), AnalyzeRequest{})
	if err == nil {
		t.Fatalf("expected send error")
	}
}
