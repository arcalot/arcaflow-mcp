## Python Analysis Engine

This directory contains the Python analysis service for workflow result analysis and optimization suggestions.

## Components

- **Result parsing** - Load and parse JSON, YAML, and log outputs
- **Metrics extraction** - Extract performance metrics from results
- **Analysis and comparison** - Multi-run comparison and statistical analysis
- **Suggestion generation** - AI-driven optimization recommendations
- **Historical database** - SQLite/PostgreSQL storage for result history
- **HTTP API** - RESTful API for Go server integration
- **gRPC interface** - gRPC service (planned, HTTP currently used)

## Documentation

### For Users
- [Result Analysis Guide](../docs/arcaflow-mcp/usage/result-analysis.md) - Using analysis features
- [Configuration Reference](../docs/arcaflow-mcp/usage/configuration.md) - Configuration options

### For Developers
- [Architecture Overview](../docs/architecture/python-engine.md) - Python engine internals
- [Development Setup](../docs/development/setup.md) - Environment setup
- [Testing Guide](../docs/development/testing.md) - Testing standards
- [API Documentation](../docs/api/python-engine.md) - Python API reference

## Quick Start

```bash
# Install dependencies
poetry install

# Run tests
poetry run pytest

# Start HTTP server
poetry run python -m arcaflow_analysis.server.app --http-address 127.0.0.1:8081
```

### gRPC service definition

The analysis service protobuf definition lives in `api/proto/analysis.proto`.
Generate Python stubs with:

```
cd analysis
poetry run python -m grpc_tools.protoc \
  -I ../api/proto \
  --python_out ../api/generated/python \
  --grpc_python_out ../api/generated/python \
  ../api/proto/analysis.proto
```

### HTTP analysis endpoint

Set `ARCAFLOW_ANALYSIS_HTTP_ADDRESS` (for example, `127.0.0.1:8081`) to enable
the HTTP endpoint used by the Go server integration. The container image
defaults this to `0.0.0.0:8081` for convenience; override it at runtime if you
need a different port or bind address.

- `GET /healthz` for health checks
- `POST /analysis/summary` for analysis requests
- `GET /analysis/history` to list stored runs
- `GET /analysis/history/{run_id}` to fetch a stored run
- `POST /analysis/history` to store a run in history

Set `ARCAFLOW_ANALYSIS_DB_URL` to enable history storage (for example,
`sqlite:////tmp/analysis-history.db`).

### CLI usage

The analysis service accepts standard help output and optional flags:

```
python -m arcaflow_analysis.server.app --help
```

Flags:

- `--http-address` (host:port) to enable the HTTP endpoint.
- `--log-level` (`debug`, `info`, `warn`, `error`) to set logging level.
- `--debug` to enable debug logging quickly.

#### POST /analysis/summary

Request body (JSON):

```
{
  "results": [
    {
      "format": "json",
      "payload": {
        "success": true,
        "metrics": {"latency_ms": 12.3},
        "records": [{"name": "sample", "value": 1}]
      }
    },
    {
      "format": "yaml",
      "payload": "success: false\nmetrics:\n  latency_ms: 18.9\n"
    }
  ],
  "compare": true,
  "metric_directions": {
    "latency_ms": "lower"
  }
}
```

Response body (JSON):

```
{
  "analysis": {
    "success_rate": 0.5,
    "metric_stats": {
      "latency_ms": {"mean": 15.6, "min": 12.3, "max": 18.9, "p95": 18.9}
    },
    "record_count": 1,
    "findings": [
      {
        "severity": "warning",
        "message": "High variability in latency_ms",
        "metric": "latency_ms",
        "value": 18.9,
        "details": {"coefficient_of_variation": 0.2}
      }
    ]
  },
  "comparison": {
    "metric_stats": {
      "latency_ms": {"mean": 15.6, "min": 12.3, "max": 18.9, "p95": 18.9}
    },
    "rankings": {
      "latency_ms": [["run-0", 12.3], ["run-1", 18.9]]
    },
    "findings": []
  },
  "suggestions": [
    {
      "title": "Reduce variability for latency_ms",
      "rationale": "High p95 relative to mean suggests variability.",
      "priority": "medium",
      "suggested_change": {"metric": "latency_ms", "target": "stability"}
    }
  ]
}
```

Notes:
- `results` is required and each entry must include `format` (`json`, `yaml`, or
  `log`) plus `payload` (raw JSON object, YAML string, or log text).
- `compare` toggles multi-run comparison output.
- `metric_directions` controls ranking (`lower` for latency, `higher` for
  throughput).

#### GET /analysis/history

Optional query parameters:

- `workflow_id` filters history by workflow.

#### GET /analysis/history/{run_id}

Fetch a stored run record by its ID.

#### POST /analysis/history

Request body (JSON):

```
{
  "workflow_id": "workflow-123",
  "input_payload": {"param": "value"},
  "metrics": {"latency_ms": 12.3}
}
```
