## Input construction tools

This section will document the input construction tools and schemas.

### `workflow_list`

Lists workflows available from a specified source. Use this tool to discover
workflow IDs and metadata before loading or describing a workflow.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
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
    "location": "/workflows"
  }
}
```

Example response:

```json
{
  "source": {
    "kind": "filesystem",
    "location": "/workflows"
  },
  "workflows": [
    {
      "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
      "name": "perf-test",
      "path": "perf-test.yaml",
      "content_sha256": "fd2b8898c0b8d9f7f5f69f23f35a9b7a1a0c4e66b8f0b8f5f13c7c6bd1d6e1df",
      "size_bytes": 1241,
      "modified_at": "2026-01-22T12:00:00Z"
    }
  ]
}
```

### `workflow_load`

Loads a specific workflow document from a source. Use this after
`workflow_list` when multiple workflows are available. If the source contains
more than one workflow, you can omit the selector when a single `workflow.yaml`
or `workflow.yml` is present, or when one exists at the shallowest path (for
example, `workflow.yaml` in the repository root). Otherwise provide
`selector.id` or `selector.path`.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  }
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    },
    "content": "version: v0.1\nsteps: {}\n",
    "content_sha256": "fd2b8898c0b8d9f7f5f69f23f35a9b7a1a0c4e66b8f0b8f5f13c7c6bd1d6e1df",
    "size_bytes": 1241,
    "modified_at": "2026-01-22T12:00:00Z"
  }
}
```

### `workflow_schema_get`

Retrieves input/output schemas from a workflow document. Input schemas come from
`input` and are resolved against sub-workflows and plugin schemas referenced by
Arcaflow namespaces. Output schemas come from `output`/`outputs` with optional
`outputSchema` refinement.

Schema resolution may execute plugin containers to fetch their input schemas. If
no container runtime is available, schema resolution will fail with a clear
error describing the missing runtime.

Namespace resolution expects step input references such as
`$.steps.<step>.starting.inputs.input` for plugin steps or
`$.steps.<step>.execute.inputs.items.item` for sub-workflow steps.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  }
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "input_json_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object"
  },
  "output_json_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object"
  },
  "example_input": {
    "sample": "value"
  },
  "schema_keys": {
    "input_key": "input",
    "output_key": "outputs"
  }
}
```

### `workflow_describe`

Provides a human-readable summary of a workflow, including high-level metadata
and a step overview.

If the source contains multiple workflows, provide `selector.id` or
`selector.path`. When a single `workflow.yaml` or `workflow.yml` exists, or a
single shallowest `workflow.yaml`/`workflow.yml` is found, the tool selects it
automatically. If not, the tool error message includes available workflow paths
to help you pick the right selector.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  }
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "version": "v0.2.0",
  "description": "Example workflow",
  "steps": [
    {
      "id": "step-a",
      "plugin_schema_ref": "plugin-schema.yaml"
    }
  ],
  "step_count": 1
}
```

### `plugin_schema_get`

Fetches plugin schemas referenced by workflow steps. This tool reads explicit
schema references (`plugin_schema_ref`, `plugin_schema`, `plugin.schema_ref`,
or `plugin.schema`) and returns the resolved schema payloads. Relative schema
paths are resolved from the workflow file location.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
    },
    "step_id": {
      "type": "string",
      "description": "Optional workflow step ID to filter results."
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  },
  "step_id": "step-a"
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "schemas": [
    {
      "step_id": "step-a",
      "location": "plugin-schema.yaml",
      "schema": {
        "type": "object",
        "properties": {
          "input": {
            "type": "string"
          }
        }
      }
    }
  ]
}
```

### `workflow_input_build`

Builds or updates a draft workflow input payload, storing it in a session and
optionally validating it against the workflow input schema.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
    },
    "session_id": {
      "type": "string",
      "description": "Optional session ID for iterative input building."
    },
    "input": {
      "type": "object",
      "description": "Partial or full workflow input payload."
    },
    "merge": {
      "type": "boolean",
      "description": "Merge input into existing draft when true.",
      "default": true
    },
    "validate": {
      "type": "boolean",
      "description": "Validate the draft input against the workflow schema.",
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  },
  "session_id": "session-1",
  "input": {
    "name": "arcaflow"
  }
}
```

Example response:

```json
{
  "session_id": "session-1",
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "draft_input": {
    "name": "arcaflow"
  },
  "validation": {
    "performed": true,
    "valid": true
  }
}
```

### `workflow_input_validate`

Validates a workflow input payload or an existing draft input session against
the workflow input schema.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
    },
    "session_id": {
      "type": "string",
      "description": "Optional session ID with stored draft input."
    },
    "input": {
      "type": "object",
      "description": "Optional workflow input payload to validate."
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  },
  "input": {
    "name": "arcaflow"
  }
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "validation": {
    "performed": true,
    "valid": true
  }
}
```

### `workflow_input_export`

Validates and exports workflow inputs as deterministic JSON or YAML payloads.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
    },
    "session_id": {
      "type": "string",
      "description": "Optional session ID with stored draft input."
    },
    "input": {
      "type": "object",
      "description": "Optional workflow input payload to export."
    },
    "format": {
      "type": "string",
      "description": "Export format: json or yaml.",
      "default": "json"
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  },
  "input": {
    "name": "arcaflow"
  },
  "format": "json"
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "format": "json",
  "payload": "{\"name\":\"arcaflow\"}",
  "metadata": {
    "workflow_id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "input_key": "input",
    "validated": true,
    "validated_at": "2026-01-22T12:00:00Z"
  }
}
```

### `workflow_input_examples_get`

Returns an example input payload generated from the workflow input schema.

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
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
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
    "location": "/workflows"
  },
  "selector": {
    "path": "perf-test.yaml"
  }
}
```

Example response:

```json
{
  "workflow": {
    "id": "b87f7e7e0f8b1d4b5b0d4b1f8d1f2cb5e1a8c7d0e14f1f2e2d4c3b5a6f7e8d9c",
    "name": "perf-test",
    "path": "perf-test.yaml",
    "source": {
      "kind": "filesystem",
      "location": "/workflows"
    }
  },
  "example_input": {
    "name": "arcaflow"
  },
  "generated": true,
  "input_key": "input"
}
```
