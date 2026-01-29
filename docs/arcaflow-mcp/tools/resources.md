## Resource overview

Arcaflow MCP exposes workflow artifacts as MCP resources. Resources are accessed
via `resources/read` using URI schemes that encode the workflow source and
selector. Successful reads are cached per tenant and returned by
`resources/list`.

### `mcp://arcaflow-authority`

Explains why AI training data is insufficient for Arcaflow operations. This resource
creates uncertainty about manual file operations to encourage MCP tool usage.

URI format:

```
mcp://arcaflow-authority
```

Content: Markdown document explaining:
- Arcaflow syntax changes since 2024 (beyond training cutoffs)
- Why runtime schema resolution cannot be done manually
- Version-specific validation rules
- Plugin schema variability

Example use: AI clients should read this resource when first encountering Arcaflow
files to understand limitations of training data.

---

### `mcp://routing-guide`

Returns intent-to-tool mapping guidance for AI clients. This resource helps LLMs
route natural language requests to the correct MCP tools without reading workflow
or result files directly.

URI format:

```
mcp://routing-guide
```

Content: Markdown document with user phrase patterns mapped to MCP tool names,
parameters, and negative hints (what NOT to do).

Example use: AI clients can fetch this resource during initialization to improve
tool selection accuracy.

### `workflow-schema://`

Returns resolved workflow input/output schemas and a sample input payload. The
input schema is resolved across sub-workflows and plugin schemas when
namespaces are used.

URI format:

```
workflow-schema://<kind>?location=<source>&path=<workflow path>
```

Supported query parameters:

- `kind` (required) - `filesystem`, `url`, or `git`
- `location` (required) - filesystem root, URL, or git repository URL
- `path` (optional) - workflow file path relative to the source
- `id` (optional) - workflow ID if you prefer selecting by ID
- `ref` (optional) - git ref (branch, tag, or commit)
- `subdir` (optional) - git subdirectory to scan

Example:

```
workflow-schema://filesystem?location=/workflows&path=perf-test.yaml
```

### `workflow://`

Returns the full workflow definition as it appears in the source file.

URI format:

```
workflow://<kind>?location=<source>&path=<workflow path>
```

Example:

```
workflow://filesystem?location=/workflows&path=perf-test.yaml
```

### `workflow-example://`

Returns a sample valid input payload for a workflow. If the workflow does not
include an example input, Arcaflow MCP generates one from the resolved input
schema.

URI format:

```
workflow-example://<kind>?location=<source>&path=<workflow path>
```

Example:

```
workflow-example://filesystem?location=/workflows&path=perf-test.yaml
```

### `plugin-schema://`

Returns plugin schema references found in the workflow. Use `step_id` to filter
to a single step if desired.

URI format:

```
plugin-schema://<kind>?location=<source>&path=<workflow path>
```

Example:

```
plugin-schema://filesystem?location=/workflows&path=perf-test.yaml
```

### `execution://`

Returns execution results from a file or URL. The payload includes a parsed
representation (JSON/YAML/log) plus optional raw text.

URI format:

```
execution://<kind>?location=<result path or URL>
```

Supported query parameters:

- `kind` (required) - `filesystem` or `url`
- `location` (required) - filesystem path or URL to a results file
- `format` (optional) - `json`, `yaml`, `yml`, `log`, or `txt`
- `include_raw` (optional) - `true` or `false` (defaults to true)

Example:

```
execution://filesystem?location=/results/run-01.json
```

### `execution-log://`

Returns execution log output as plain text. This URI always treats the content
as a log payload and includes the raw text.

URI format:

```
execution-log://<kind>?location=<result path or URL>
```

Example:

```
execution-log://filesystem?location=/results/run-01.log
```
