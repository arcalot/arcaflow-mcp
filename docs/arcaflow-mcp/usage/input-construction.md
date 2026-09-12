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

Arcaflow derives input schemas from the workflow `input` section and output
schemas from either the `output` or `outputs` section. An optional
`outputSchema` section is available for user-refinement of the output schema.
Input schema resolution follows Arcaflow namespaces, resolving sub-workflow and
plugin schemas when refs point into step inputs. Input validation uses the
Arcaflow plugin SDK schema definitions, matching the engine behavior.

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

### Plugin schema resolution for namespaced refs

When workflows reference plugin input schemas using Arcaflow namespace refs
(for example, `$.steps.<step>.starting.inputs.input`), the server resolves the
schema by executing the plugin container with `--json-schema input`. A container
runtime (Podman or Docker) is required for this resolution step.

### State management

The server tracks input construction sessions per tenant. Draft inputs and
metadata are isolated per tenant and expire after a configurable TTL. This
prevents cross-tenant leakage while enabling iterative input building.

### Input file generation

Generated input files are emitted only after validation succeeds. The generator
returns the payload in JSON or YAML plus metadata confirming validation (workflow
ID, schema path, timestamp), while the file contents remain Arcaflow-compatible
input payloads.
