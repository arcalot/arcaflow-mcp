## Plugin discovery tools

Plugin discovery tools enable AI agents and external consumers
to browse the Arcaflow plugin catalog and inspect plugin
capabilities without needing direct access to container
registries or a container runtime on the client host.

### plugin_list

List available Arcaflow plugins from Quay.io registries with
enriched metadata including keywords, categories, step info,
and supported architectures.

**When to use:**
- "What plugins are available?"
- "List storage plugins"
- "What benchmarks can I run on arm64?"

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "category": {
      "type": "string",
      "description": "Filter by category (e.g., storage, cpu, network, stress)."
    },
    "architecture": {
      "type": "string",
      "description": "Filter by architecture (e.g., amd64, arm64)."
    }
  },
  "additionalProperties": false
}
```

Both filters are optional. Omit them to return all plugins.

**Output fields:**

| Field | Type | Description |
|-------|------|-------------|
| `plugins` | array | List of plugin metadata objects |
| `plugins[].name` | string | Repository name (e.g., `arcaflow-plugin-fio`) |
| `plugins[].image` | string | Full image reference without tag |
| `plugins[].version` | string | Latest semver tag from Quay |
| `plugins[].description` | string | Human-readable description |
| `plugins[].keywords` | string[] | Search/matching keywords |
| `plugins[].architectures` | string[] | Supported platforms |
| `plugins[].category` | string | Primary classification |
| `plugins[].default_step` | string | Default step ID |
| `plugins[].steps` | string[] | All available step IDs |
| `total` | int | Total plugins before filtering |
| `filtered` | int | Plugins after filtering |

**Caching:** The plugin catalog is cached in memory with a
configurable TTL (default 1 hour, set via `--plugin-cache-ttl`
CLI flag). If Quay.io is unreachable, stale cache data is
returned. Only one Quay fetch runs at a time to avoid rate
limits.

**Metadata sources:** Keywords, categories, steps, and
architectures come from an embedded metadata catalog
(`server/config/plugin_metadata.yaml`). Plugins not in
the catalog get inferred metadata: keywords derived from
the repo name, category `"other"`, default step `"workload"`,
architectures `["unknown"]`.

**Registry sources:** Scans `arcalot` and `redhat-performance`
organisations on Quay.io. Base images, templates, and test
scaffolds are excluded.

---

### plugin_describe

Get detailed information about a specific Arcaflow plugin
including its full step schemas (inputs and outputs).

**When to use:**
- "Show me the fio plugin schema"
- "What inputs does uperf need?"
- "Describe the pcp plugin steps"

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "plugin": {
      "type": "string",
      "description": "Plugin name or full image reference."
    },
    "version": {
      "type": "string",
      "description": "Version tag. Defaults to 'latest'."
    }
  },
  "required": ["plugin"],
  "additionalProperties": false
}
```

The `plugin` field accepts either a bare name
(`arcaflow-plugin-fio`) or a full image reference
(`quay.io/arcalot/arcaflow-plugin-fio`).

**Output fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Plugin repository name |
| `image` | string | Full image:tag reference |
| `description` | string | From step display metadata or name |
| `architectures` | string[] | Supported platforms |
| `steps` | object | Map of step ID → step info |
| `steps[].id` | string | Step identifier |
| `steps[].display` | object | Optional name and description |
| `steps[].input_schema` | object | Arcaflow input scope schema |
| `steps[].outputs` | object | Arcaflow output schemas |
| `default_step` | string | Default step (if single-step plugin) |
| `schemas_available` | bool | Whether schemas were retrieved |
| `schema_raw` | object | Full parsed Arcaflow schema |

**Schema retrieval:** Runs
`podman/docker run --rm <image> --schema` to get the complete
Arcaflow plugin schema as YAML. This works for both Go and
Python SDK plugins.

- Requires podman or docker on the MCP server host
- If no container runtime is available, returns metadata
  without schemas (`schemas_available: false`)
- First call for a new image may be slow due to image pull
- 30-second timeout on container execution

**Security:** Only images from allowed registries are executed:

- `quay.io/arcalot/`
- `quay.io/redhat-performance/`

Images from other registries are rejected with an error.
Prefix validation prevents similar-org attacks
(e.g., `quay.io/arcalot-evil/` is rejected).

**Schema caching:** Schemas for tagged versions (e.g., `0.9.0`)
are cached indefinitely — immutable tags produce identical
schemas. The `latest` tag is never cached since it can change.

**Schema format:** The `schema_raw` field contains the full
Arcaflow schema in its native format (not JSON Schema). This
includes type definitions, object scopes, enum constraints,
and all step I/O contracts. Consumers that need JSON Schema
must convert from the Arcaflow type system.

---

### Examples

**Browse all plugins:**
```
→ plugin_list({})
← { "plugins": [...], "total": 17, "filtered": 17 }
```

**Find storage benchmarks:**
```
→ plugin_list({"category": "storage"})
← { "plugins": [{"name": "arcaflow-plugin-fio", ...}], "total": 17, "filtered": 1 }
```

**Filter by architecture:**
```
→ plugin_list({"architecture": "arm64"})
← { "plugins": [...], "total": 17, "filtered": 12 }
```

**Get plugin schema:**
```
→ plugin_describe({"plugin": "arcaflow-plugin-fio", "version": "0.9.0"})
← {
    "name": "arcaflow-plugin-fio",
    "image": "quay.io/arcalot/arcaflow-plugin-fio:0.9.0",
    "schemas_available": true,
    "steps": {
      "workload": {
        "id": "workload",
        "display": {"name": "Run workload", ...},
        "input_schema": {...},
        "outputs": {...}
      }
    },
    "default_step": "workload",
    "schema_raw": {...}
  }
```

**No container runtime available:**
```
→ plugin_describe({"plugin": "arcaflow-plugin-fio"})
← {
    "name": "arcaflow-plugin-fio",
    "schemas_available": false,
    "steps": {}
  }
```
