## Result analysis tools

This section will document the output analysis tools and schemas.

### `workflow_results_load`

Loads workflow result files from disk or URL and detects JSON/YAML/log content.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "source": {
      "type": "object",
      "properties": {
        "kind": {
          "type": "string",
          "description": "Result source kind: filesystem or url."
        },
        "location": {
          "type": "string",
          "description": "Filesystem path or URL for the result file."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "format": {
      "type": "string",
      "description": "Optional format hint: json, yaml, yml, log, or txt."
    },
    "include_raw": {
      "type": "boolean",
      "description": "Include raw text in the response when true.",
      "default": true
    }
  },
  "required": ["source"],
  "additionalProperties": false
}
```

Example request:

```json
{
  "source": {
    "kind": "filesystem",
    "location": "/path/to/result.json"
  }
}
```

Example response:

```json
{
  "source": {
    "kind": "filesystem",
    "location": "/path/to/result.json"
  },
  "format": "json",
  "payload": {
    "success": true
  },
  "raw_text": "{\"success\":true}",
  "size_bytes": 17,
  "content_sha256": "5b8df5e53130d4280d2c0c2dfb7f6f57c0c0336a37d1acaa2a3b52dc269f7b9c"
}
```

### `workflow_results_parse`

Parses result payloads and returns summary statistics.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 1
    }
  },
  "required": ["results"],
  "additionalProperties": false
}
```

Example request:

```json
{
  "results": [
    {
      "format": "json",
      "payload": {"success": true}
    }
  ]
}
```

Example response:

```json
{
  "analysis": {
    "success_rate": 1.0,
    "metric_stats": {},
    "record_count": 1,
    "findings": []
  }
}
```

### `workflow_results_analyze`

Analyzes results and returns suggestions for improving inputs.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 1
    }
  },
  "required": ["results"],
  "additionalProperties": false
}
```

Example response:

```json
{
  "analysis": {
    "success_rate": 0.9,
    "metric_stats": {},
    "record_count": 10,
    "findings": []
  },
  "suggestions": [
    {"title": "Tune input", "priority": "high", "rationale": "Reduce variance."}
  ]
}
```

### `workflow_results_compare`

Compares multiple runs and ranks metrics.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 2
    },
    "metric_directions": {
      "type": "object",
      "description": "Map of metric names to higher or lower.",
      "additionalProperties": {
        "type": "string"
      }
    }
  },
  "required": ["results"],
  "additionalProperties": false
}
```

Example response:

```json
{
  "analysis": {
    "success_rate": 1.0,
    "metric_stats": {},
    "record_count": 2,
    "findings": []
  },
  "comparison": {
    "metric_stats": {},
    "rankings": {},
    "findings": []
  }
}
```

### `workflow_inputs_suggest`

Returns input modification suggestions derived from analysis.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 1
    }
  },
  "required": ["results"],
  "additionalProperties": false
}
```

Example response:

```json
{
  "suggestions": [
    {"title": "Tune input", "priority": "high", "rationale": "Reduce variance."}
  ]
}
```

### `workflow_optimization_guide`

Summarizes strategic optimization guidance from results.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 1
    }
  },
  "required": ["results"],
  "additionalProperties": false
}
```

### `workflow_results_metrics_extract`

Extracts metric statistics from result payloads.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 1
    }
  },
  "required": ["results"],
  "additionalProperties": false
}
```

Example response:

```json
{
  "metric_stats": {
    "latency_ms": {"mean": 10.0, "p95": 12.0}
  },
  "record_count": 2
}
```

### `workflow_history_load`

Loads stored analysis history summaries or a single run record.

History access requires the analysis service to be configured with
`ARCAFLOW_ANALYSIS_DB_URL`.

Input schema:

```json
{
  "type": "object",
  "properties": {
    "workflow_id": {
      "type": "string",
      "description": "Optional workflow ID to filter history."
    },
    "run_id": {
      "type": "string",
      "description": "Optional run ID to fetch a specific record."
    }
  },
  "additionalProperties": false
}
```

Example response (list):

```json
{
  "runs": [
    {
      "run_id": "run-1",
      "workflow_id": "workflow-1",
      "created_at": "2026-01-23T00:00:00Z",
      "metrics": {"latency_ms": 12.3}
    }
  ]
}
```

Example response (single):

```json
{
  "run": {
    "run_id": "run-1",
    "workflow_id": "workflow-1",
    "created_at": "2026-01-23T00:00:00Z",
    "input_payload": {"param": "value"},
    "metrics": {"latency_ms": 12.3}
  }
}
```

Example response:

```json
{
  "guidance": "Prioritize the highest impact suggestions, then validate improvements.",
  "suggestions": [
    {"title": "Tune input", "priority": "high", "rationale": "Reduce variance."}
  ]
}
```
