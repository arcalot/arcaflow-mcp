package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ResultPayload describes a result payload sent to the analysis service.
type ResultPayload struct {
	Format  string          `json:"format"`
	Payload json.RawMessage `json:"payload"`
}

// AnalyzeRequest holds analysis request parameters.
type AnalyzeRequest struct {
	Results          []ResultPayload   `json:"results"`
	Compare          bool              `json:"compare,omitempty"`
	MetricDirections map[string]string `json:"metric_directions,omitempty"`
}

// AnalysisFinding describes an analysis finding from the service.
type AnalysisFinding struct {
	Severity string                 `json:"severity"`
	Message  string                 `json:"message"`
	Metric   string                 `json:"metric,omitempty"`
	Value    float64                `json:"value,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// AnalysisSummary describes the analysis summary payload.
type AnalysisSummary struct {
	SuccessRate *float64                      `json:"success_rate"`
	MetricStats map[string]map[string]float64 `json:"metric_stats"`
	RecordCount int                           `json:"record_count"`
	Findings    []AnalysisFinding             `json:"findings"`
}

// ComparisonSummary describes the comparison summary payload.
type ComparisonSummary struct {
	MetricStats map[string]map[string]float64 `json:"metric_stats"`
	Rankings    map[string][][]interface{}    `json:"rankings"`
	Findings    []AnalysisFinding             `json:"findings"`
}

// AnalyzeResponse wraps analysis, comparison, and suggestions.
type AnalyzeResponse struct {
	Analysis    AnalysisSummary          `json:"analysis"`
	Comparison  *ComparisonSummary       `json:"comparison,omitempty"`
	Suggestions []map[string]interface{} `json:"suggestions"`
}

// HistoryRunSummary describes a stored run summary.
type HistoryRunSummary struct {
	RunID      string                 `json:"run_id"`
	WorkflowID string                 `json:"workflow_id"`
	CreatedAt  string                 `json:"created_at"`
	Metrics    map[string]interface{} `json:"metrics"`
}

// HistoryRunRecord describes a stored run record.
type HistoryRunRecord struct {
	RunID        string                 `json:"run_id"`
	WorkflowID   string                 `json:"workflow_id"`
	CreatedAt    string                 `json:"created_at"`
	InputPayload map[string]interface{} `json:"input_payload"`
	Metrics      map[string]interface{} `json:"metrics"`
}

// Client communicates with the analysis HTTP API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient constructs a new analysis client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Analyze submits results to the analysis service.
func (client *Client) Analyze(
	ctx context.Context,
	request AnalyzeRequest,
) (AnalyzeResponse, error) {
	if ctx.Err() != nil {
		return AnalyzeResponse{}, ctx.Err()
	}
	if client.baseURL == "" {
		return AnalyzeResponse{}, fmt.Errorf("analysis base url is required")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return AnalyzeResponse{}, fmt.Errorf("marshal analysis request: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.baseURL+"/analysis/summary",
		bytes.NewReader(body),
	)
	if err != nil {
		return AnalyzeResponse{}, fmt.Errorf("build analysis request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return AnalyzeResponse{}, fmt.Errorf("send analysis request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return AnalyzeResponse{}, fmt.Errorf("analysis service status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return AnalyzeResponse{}, fmt.Errorf("read analysis response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return AnalyzeResponse{}, fmt.Errorf("close analysis response: %w", err)
	}
	var response AnalyzeResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return AnalyzeResponse{}, fmt.Errorf("decode analysis response: %w", err)
	}
	return response, nil
}

// HistoryList fetches stored run summaries from the analysis service.
func (client *Client) HistoryList(
	ctx context.Context,
	workflowID string,
) ([]HistoryRunSummary, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if client.baseURL == "" {
		return nil, fmt.Errorf("analysis base url is required")
	}
	endpoint := client.baseURL + "/analysis/history"
	if strings.TrimSpace(workflowID) != "" {
		endpoint = endpoint + "?workflow_id=" + url.QueryEscape(workflowID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build history request: %w", err)
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send history request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("analysis service status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read history response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("close history response: %w", err)
	}
	var response struct {
		Runs []HistoryRunSummary `json:"runs"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode history response: %w", err)
	}
	return response.Runs, nil
}

// HistoryGet fetches a stored run record by ID.
func (client *Client) HistoryGet(
	ctx context.Context,
	runID string,
) (HistoryRunRecord, error) {
	if ctx.Err() != nil {
		return HistoryRunRecord{}, ctx.Err()
	}
	if client.baseURL == "" {
		return HistoryRunRecord{}, fmt.Errorf("analysis base url is required")
	}
	if strings.TrimSpace(runID) == "" {
		return HistoryRunRecord{}, fmt.Errorf("run id is required")
	}
	endpoint := client.baseURL + "/analysis/history/" + url.PathEscape(runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return HistoryRunRecord{}, fmt.Errorf("build history request: %w", err)
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return HistoryRunRecord{}, fmt.Errorf("send history request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return HistoryRunRecord{}, fmt.Errorf("analysis service status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return HistoryRunRecord{}, fmt.Errorf("read history response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return HistoryRunRecord{}, fmt.Errorf("close history response: %w", err)
	}
	var response struct {
		Run HistoryRunRecord `json:"run"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return HistoryRunRecord{}, fmt.Errorf("decode history response: %w", err)
	}
	return response.Run, nil
}
