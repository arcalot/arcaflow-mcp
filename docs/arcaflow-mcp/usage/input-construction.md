## Input construction

This guide will show how to load workflows, extract schemas, build inputs, and
validate before export.

### Workflow discovery sources

Workflow discovery currently supports these sources:

- Filesystem paths (files or directories, scanned recursively)
- HTTP/HTTPS URLs (single workflow per URL)
- Git repositories (ref + optional subdirectory)

Only `.yaml`, `.yml`, and `.json` files are treated as workflows. For directory
and git sources, results are sorted by relative path to keep discovery
deterministic.

### Metadata captured for each workflow

Each discovered workflow includes metadata that is reused in later stages:

- Source type, location, ref, and subdirectory (when applicable)
- Path relative to the source root
- Content SHA-256 hash (stable ID seed)
- File size and modification time (filesystem and git)
- Fetch time, ETag, and Last-Modified headers (HTTP sources)

These metadata fields support caching, repeatable schema extraction, and stable
workflow identifiers for downstream validation and export.

### Workflow parsing and schema extraction

Arcaflow workflows describe input schemas under the `input` key. If explicit
`input_schema`/`output_schema` entries are present they are used; otherwise the
parser falls back to the `input` and `outputs` sections. Input validation uses
the Arcaflow plugin SDK schema definitions, matching the engine behavior.

If the input schema is missing or invalid, parsing fails with a clear error so
invalid workflows never progress to the validation or export steps.

### Input validation behavior

Input payloads are parsed as JSON or YAML, then validated with
`schema.DescribeScope().Unserialize` to ensure they match the workflow input
schema. Validated inputs are re-serialized for deterministic JSON output.

### Plugin schema references

When workflows include plugin schema references, the server caches them for
future validation and documentation. Supported workflow step keys:

- `plugin_schema_ref` or `plugin_schema`
- `plugin.schema_ref` or `plugin.schema`

Relative paths are resolved from the workflow file location, and HTTP(S) URLs
are fetched directly.

### State management

The server tracks input construction sessions per tenant. Draft inputs and
metadata are isolated per tenant and expire after a configurable TTL. This
prevents cross-tenant leakage while enabling iterative input building.

### Input file generation

Generated input files are emitted only after validation succeeds. The generator
returns the payload in JSON or YAML plus metadata confirming validation (workflow
ID, schema path, timestamp), while the file contents remain Arcaflow-compatible
input payloads.
