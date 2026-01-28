# Inter-Service Protocol Reference

API protocol documentation for communication between the Go MCP server and Python analysis engine.

## Current Implementation: HTTP REST

### Overview

The Go server and Python analysis engine currently communicate via **HTTP REST API**. This provides a simple, language-agnostic protocol that's easy to debug and monitor.

**Protocol:**
- HTTP/1.1 or HTTP/2
- JSON request/response payloads
- RESTful endpoint design

**Transport:**
- Default: `http://localhost:8081`
- Configurable via `ARCAFLOW_MCP_ANALYSIS_HTTP_URL`

### API Specification

For complete HTTP API specification, see:

- [Inter-Service Communication](../architecture/inter-service.md) - Detailed protocol documentation
- [Python Analysis Engine API](python-engine.md) - Python endpoint implementations
- [Go Server API](go-server.md) - Go client implementation

### HTTP Endpoints Summary

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health check |
| `/analyze` | POST | Analyze single result |
| `/compare` | POST | Compare multiple results |
| `/suggest` | POST | Generate suggestions |
| `/metrics` | POST | Extract metrics only |
| `/parse` | POST | Parse result |

### Request/Response Format

**Common Request Structure:**

```json
{
  "results": [
    {
      "payload": { ... },
      "format": "json"
    }
  ],
  "metric_directions": {
    "metric_name": "higher" | "lower"
  },
  "compare": true | false
}
```

**Common Response Structure:**

```json
{
  "metrics": { ... },
  "suggestions": [ ... ],
  "trends": { ... },
  "anomalies": [ ... ],
  "comparison": { ... }
}
```

### Error Handling

**Error Response Format:**

```json
{
  "error": "Error message",
  "detail": "Detailed error description",
  "code": "ERROR_CODE"
}
```

**Error Codes:**

- `VALIDATION_ERROR` - Invalid input
- `PARSE_ERROR` - Result parse failure
- `INTERNAL_ERROR` - Server error

---

## Future: gRPC Protocol

### Rationale for gRPC

gRPC may be adopted in the future for:

1. **Performance**: Binary protocol, HTTP/2 multiplexing
2. **Type Safety**: Strongly-typed protocol buffers
3. **Streaming**: Bi-directional streaming support
4. **Code Generation**: Auto-generated clients/servers

### Proposed Service Definition

```protobuf
// analysis.proto
syntax = "proto3";

package arcaflow.analysis.v1;

import "google/protobuf/struct.proto";

// Analysis service for workflow results
service AnalysisService {
  // Health check
  rpc Health(HealthRequest) returns (HealthResponse);
  
  // Analyze a single result
  rpc Analyze(AnalyzeRequest) returns (AnalyzeResponse);
  
  // Compare multiple results
  rpc Compare(CompareRequest) returns (CompareResponse);
  
  // Generate suggestions
  rpc Suggest(SuggestRequest) returns (SuggestResponse);
  
  // Extract metrics only
  rpc ExtractMetrics(MetricsRequest) returns (MetricsResponse);
  
  // Parse result
  rpc Parse(ParseRequest) returns (ParseResponse);
  
  // Stream analysis results (future)
  rpc StreamAnalyze(stream AnalyzeRequest) returns (stream AnalyzeResponse);
}

// Health check request
message HealthRequest {}

// Health check response
message HealthResponse {
  string status = 1;  // "healthy" or "unhealthy"
  string version = 2;
  int64 uptime_seconds = 3;
}

// Analyze request
message AnalyzeRequest {
  repeated Result results = 1;
  map<string, string> metric_directions = 2;  // metric -> "higher"|"lower"
  bool compare = 3;
}

// Result payload
message Result {
  google.protobuf.Struct payload = 1;  // Result data
  string format = 2;                   // "json", "yaml", "txt"
}

// Analyze response
message AnalyzeResponse {
  map<string, MetricStats> metrics = 1;
  repeated Suggestion suggestions = 2;
  map<string, string> trends = 3;
  repeated string anomalies = 4;
  Comparison comparison = 5;  // Optional
}

// Metric statistics
message MetricStats {
  double min = 1;
  double max = 2;
  double mean = 3;
  double median = 4;
  double p95 = 5;
  double p99 = 6;
  double std_dev = 7;
  double cv = 8;
  int32 count = 9;
}

// Optimization suggestion
message Suggestion {
  string category = 1;       // "performance", "configuration", "resource"
  string priority = 2;       // "high", "medium", "low"
  string metric = 3;
  double current = 4;
  double target = 5;
  string recommendation = 6;
  string rationale = 7;
  string expected_impact = 8;
  double confidence = 9;
}

// Comparison result
message Comparison {
  map<string, Rankings> rankings = 1;    // metric -> rankings
  map<string, Deltas> deltas = 2;        // metric -> deltas
  int32 best_overall = 3;
  string summary = 4;
}

// Rankings for a metric
message Rankings {
  repeated int32 indices = 1;
}

// Deltas for a metric
message Deltas {
  repeated double values = 1;
}

// Compare request
message CompareRequest {
  repeated Result results = 1;
  map<string, string> metric_directions = 2;
  bool compare = 3;
}

// Compare response
message CompareResponse {
  map<string, MetricStats> metrics = 1;
  Comparison comparison = 2;
  repeated Suggestion suggestions = 3;
}

// Suggest request
message SuggestRequest {
  repeated Result results = 1;
  map<string, string> metric_directions = 2;
}

// Suggest response
message SuggestResponse {
  repeated Suggestion suggestions = 1;
}

// Metrics request
message MetricsRequest {
  repeated Result results = 1;
}

// Metrics response
message MetricsResponse {
  map<string, MetricStats> metrics = 1;
}

// Parse request
message ParseRequest {
  repeated Result results = 1;
}

// Parse response
message ParseResponse {
  ParsedResult parsed = 1;
}

// Parsed result
message ParsedResult {
  map<string, double> metrics = 1;
  google.protobuf.Struct metadata = 2;
  google.protobuf.Struct raw = 3;
  string format = 4;
  repeated string errors = 5;
}
```

### Code Generation

**Go Client:**

```bash
# Install protoc and Go plugin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       analysis.proto
```

**Python Server:**

```bash
# Install grpcio-tools
poetry add grpcio-tools

# Generate Python code
python -m grpc_tools.protoc -I. \
  --python_out=. \
  --grpc_python_out=. \
  analysis.proto
```

### Client Usage (Go)

```go
import (
    "context"
    "google.golang.org/grpc"
    pb "github.com/arcalot/arcaflow-mcp/server/pkg/analysis/proto"
)

// Create gRPC client
conn, err := grpc.Dial("localhost:8081", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewAnalysisServiceClient(conn)

// Call Analyze
req := &pb.AnalyzeRequest{
    Results: []*pb.Result{
        {
            Payload: &structpb.Struct{ /* ... */ },
            Format:  "json",
        },
    },
    MetricDirections: map[string]string{
        "throughput": "higher",
        "latency":    "lower",
    },
}

resp, err := client.Analyze(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

// Process response
for metric, stats := range resp.Metrics {
    log.Printf("%s: mean=%f, p95=%f", metric, stats.Mean, stats.P95)
}
```

### Server Implementation (Python)

```python
import grpc
from concurrent import futures
import analysis_pb2
import analysis_pb2_grpc

class AnalysisServicer(analysis_pb2_grpc.AnalysisServiceServicer):
    def Analyze(self, request, context):
        # Process request
        results = request.results
        metric_directions = dict(request.metric_directions)
        
        # Perform analysis
        analysis = perform_analysis(results, metric_directions)
        
        # Build response
        response = analysis_pb2.AnalyzeResponse(
            metrics={...},
            suggestions=[...],
        )
        
        return response

# Start server
server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
analysis_pb2_grpc.add_AnalysisServiceServicer_to_server(
    AnalysisServicer(), server
)
server.add_insecure_port('[::]:8081')
server.start()
server.wait_for_termination()
```

---

## Migration Path

### Phase 1: HTTP REST (Current)

- **Status**: Implemented
- **Pros**: Simple, widely supported, easy debugging
- **Cons**: Less efficient than gRPC, no type safety

### Phase 2: Dual Protocol Support (Future)

- Support both HTTP REST and gRPC
- Use feature flags or configuration to select protocol
- Gradual migration path for existing deployments

```yaml
# Configuration
analysis:
  protocol: "http"  # or "grpc"
  http_url: "http://localhost:8081"
  grpc_address: "localhost:8081"
```

### Phase 3: gRPC Primary (Long-term)

- gRPC as default protocol
- HTTP REST maintained for compatibility
- Deprecation timeline for HTTP-only mode

---

## Performance Comparison

### HTTP REST

**Pros:**
- Widely supported (curl, browsers, load balancers)
- Human-readable JSON payloads
- Easy debugging and monitoring
- No special infrastructure required

**Cons:**
- JSON serialization overhead
- HTTP/1.1 lacks multiplexing
- No compile-time type checking
- Larger payload sizes

**Benchmarks:**
- ~50-100 requests/sec per connection
- ~10ms latency (local network)
- ~1-5 MB/s throughput

### gRPC

**Pros:**
- Binary protocol (faster serialization)
- HTTP/2 multiplexing
- Compile-time type safety
- Smaller payloads
- Bi-directional streaming

**Cons:**
- Requires HTTP/2 infrastructure
- Less human-readable (binary)
- Debugging more complex
- Requires code generation

**Benchmarks (estimated):**
- ~200-500 requests/sec per connection
- ~1-5ms latency (local network)
- ~10-50 MB/s throughput

---

## Security Considerations

### Current HTTP REST

**Authentication:**
- No authentication (trusted internal network)
- Future: API keys, JWT tokens

**Encryption:**
- Plain HTTP (not secure)
- Future: TLS/HTTPS

### Future gRPC

**Authentication:**
- Mutual TLS (mTLS)
- Token-based authentication
- Per-RPC credentials

**Encryption:**
- TLS by default
- Certificate-based authentication

**Example gRPC Security:**

```go
// Server TLS config
creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
if err != nil {
    log.Fatal(err)
}

server := grpc.NewServer(grpc.Creds(creds))
```

---

## Monitoring and Observability

### HTTP REST

**Metrics:**
- Request count (by endpoint)
- Response time (by endpoint)
- Error rate (by endpoint)

**Tracing:**
- HTTP request tracing (OpenTelemetry)
- Distributed tracing (Jaeger, Zipkin)

**Tools:**
- Prometheus for metrics
- Grafana for visualization
- ELK stack for logs

### gRPC

**Metrics:**
- Per-RPC metrics (built-in)
- Latency histograms
- Streaming metrics

**Tracing:**
- gRPC tracing interceptors
- OpenTelemetry integration

**Tools:**
- grpc-go middleware (grpc_prometheus)
- grpc_opentracing
- grpc_zap (logging)

---

## Related Documentation

- [Inter-Service Communication](../architecture/inter-service.md) - Current HTTP protocol details
- [Go Server API](go-server.md) - Go client implementation
- [Python Analysis Engine API](python-engine.md) - Python server implementation

---

*For HTTP API specification, see [Inter-Service Communication](../architecture/inter-service.md).*
