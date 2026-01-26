package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

func TestHistoryListSuccess(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/analysis/history" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		response := map[string]interface{}{
			"runs": []map[string]interface{}{
				{
					"run_id":      "run-1",
					"workflow_id": "workflow-1",
					"created_at":  "2026-01-23T00:00:00Z",
					"metrics":     map[string]interface{}{"latency": 1.0},
				},
			},
		}
		_ = json.NewEncoder(writer).Encode(response)
	})

	server := newAnalysisTestServer(t, handler)
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	runs, err := client.HistoryList(context.Background(), "")
	if err != nil {
		t.Fatalf("history list failed: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run")
	}
}

func TestHistoryGetSuccess(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/analysis/history/run-1" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		response := map[string]interface{}{
			"run": map[string]interface{}{
				"run_id":      "run-1",
				"workflow_id": "workflow-1",
				"created_at":  "2026-01-23T00:00:00Z",
				"metrics":     map[string]interface{}{"latency": 1.0},
			},
		}
		_ = json.NewEncoder(writer).Encode(response)
	})

	server := newAnalysisTestServer(t, handler)
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	run, err := client.HistoryGet(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("history get failed: %v", err)
	}
	if run.RunID != "run-1" {
		t.Fatalf("expected run id run-1")
	}
}

func TestHistoryListMissingBaseURL(t *testing.T) {
	t.Parallel()

	client := NewClient("")
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for missing base url")
	}
}

func TestHistoryGetMissingRunID(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	_, err := client.HistoryGet(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for missing run id")
	}
}

func newAnalysisTestServer(
	t *testing.T,
	handler http.Handler,
) *httptest.Server {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("socket operations not permitted: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

func TestHistoryListStatusError(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	})
	server := newAnalysisTestServer(t, handler)
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected status error")
	}
}

func TestHistoryListDecodeError(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("{invalid-json"))
	})
	server := newAnalysisTestServer(t, handler)
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestHistoryGetStatusError(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	})
	server := newAnalysisTestServer(t, handler)
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	_, err := client.HistoryGet(context.Background(), "run-1")
	if err == nil {
		t.Fatalf("expected status error")
	}
}

func TestHistoryListCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("http://example.com")
	_, err := client.HistoryList(ctx, "")
	if err == nil {
		t.Fatalf("expected canceled context error")
	}
}

func TestHistoryListRequestBuildError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://[::1")
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected request build error")
	}
}

func TestHistoryListSendError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: errorRoundTripper{},
	}
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected send error")
	}
}

func TestHistoryListReadError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: responseRoundTripper{
			statusCode: http.StatusOK,
			body:       &errorBody{readErr: errors.New("read failed")},
		},
	}
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected read error")
	}
}

func TestHistoryListCloseError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: responseRoundTripper{
			statusCode: http.StatusOK,
			body: &errorBody{
				data:     []byte(`{"runs":[]}`),
				closeErr: errors.New("close failed"),
			},
		},
	}
	_, err := client.HistoryList(context.Background(), "")
	if err == nil {
		t.Fatalf("expected close error")
	}
}

func TestHistoryGetCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("http://example.com")
	_, err := client.HistoryGet(ctx, "run-1")
	if err == nil {
		t.Fatalf("expected canceled context error")
	}
}

func TestHistoryGetRequestBuildError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://[::1")
	_, err := client.HistoryGet(context.Background(), "run-1")
	if err == nil {
		t.Fatalf("expected request build error")
	}
}

func TestHistoryGetSendError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: errorRoundTripper{},
	}
	_, err := client.HistoryGet(context.Background(), "run-1")
	if err == nil {
		t.Fatalf("expected send error")
	}
}

func TestHistoryGetDecodeError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: responseRoundTripper{
			statusCode: http.StatusOK,
			body:       &errorBody{data: []byte("{invalid-json")},
		},
	}
	_, err := client.HistoryGet(context.Background(), "run-1")
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestHistoryGetCloseError(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com")
	client.httpClient = &http.Client{
		Transport: responseRoundTripper{
			statusCode: http.StatusOK,
			body: &errorBody{
				data:     []byte(`{"run":{"run_id":"run-1"}}`),
				closeErr: errors.New("close failed"),
			},
		},
	}
	_, err := client.HistoryGet(context.Background(), "run-1")
	if err == nil {
		t.Fatalf("expected close error")
	}
}

type responseRoundTripper struct {
	statusCode int
	body       io.ReadCloser
}

func (roundTripper responseRoundTripper) RoundTrip(
	*http.Request,
) (*http.Response, error) {
	return &http.Response{
		StatusCode: roundTripper.statusCode,
		Body:       roundTripper.body,
		Header:     make(http.Header),
	}, nil
}

type errorBody struct {
	readErr  error
	closeErr error
	data     []byte
	read     bool
}

func (body *errorBody) Read(dest []byte) (int, error) {
	if body.readErr != nil {
		return 0, body.readErr
	}
	if body.read {
		return 0, io.EOF
	}
	body.read = true
	n := copy(dest, body.data)
	if n < len(body.data) {
		body.data = body.data[n:]
		body.read = false
		return n, nil
	}
	return n, io.EOF
}

func (body *errorBody) Close() error {
	return body.closeErr
}
