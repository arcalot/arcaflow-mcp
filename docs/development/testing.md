## Testing

This document describes unit, integration, and MCP compliance testing for the
Arcaflow MCP server.

### Automated tests

Run Go tests for the server:

```
./scripts/test-go.sh
```

Run Go tests with coverage:

```
./scripts/test-go-coverage.sh
```

If you need to run Go tests manually, set local cache paths to avoid sandbox
permission issues:

```
cd server
GOCACHE="/path/to/.gocache" GOMODCACHE="/path/to/.gomodcache" go test ./... -count=1
```

Run Python tests for the analysis engine:

```
cd analysis
poetry run pytest
```

Run Python tests with coverage:

```
./scripts/test-python-coverage.sh
```

Run integration validation with a real Arcaflow engine and workflow:

```
./scripts/test-integration.sh
```

This integration test uses pinned versions defined in
`scripts/test-integration.sh` (the single source of truth).

The integration test requires a container runtime (`docker` in CI, or `podman`
locally) and network access to fetch the engine and workflow assets.
Ensure the container runtime daemon is running (e.g. Docker socket available)
before executing `scripts/test-integration.sh`.

Run integration tests against the target auto-perf workflow (no execution):

```
ARCAFLOW_AUTO_PERF_PATH="/path/to/arcaflow-workflow-auto-perf" \
  go test ./server/test/integration -run AutoPerf
```

`ARCAFLOW_AUTO_PERF_PATH` can point to the workflow directory or a specific
`workflow.yaml` file. The tests resolve input schemas (including namespace refs
into sub-workflows) using a stubbed plugin schema provider so they do not require
container runtime access.

Set `ARCAFLOW_MCP_PLUGIN_SCHEMA_MODE=stub` to force stub plugin schemas during
manual validation runs that exercise input validation with plugin namespaces.

### Performance benchmarks

Run benchmarks for typical input construction operations:

```
go test ./server/pkg/arcaflow/workflow -bench=. -run ^$
```

Record results in `DEVELOPMENT_PLAN.md` and validate the `<100ms` overhead target
for common operations.

### MCP compliance checklist

Use this checklist to verify MCP protocol compliance (record results in
`DEVELOPMENT_PLAN.md`):

- JSON-RPC 2.0 parsing and validation
- Request/response correlation by `id`
- Notification handling (no `id` means no response)
- Standard JSON-RPC error codes
- MCP required methods: `initialize`, `tools/list`, `tools/call`,
  `resources/list`, `resources/read`, `ping`
- Capability negotiation in `initialize`

Recommended automated coverage:

```
go test ./server/pkg/protocol ./server/pkg/transport/stdio
go test ./server/test/integration -run HTTP
```

### Phase 5 test scenarios

Record outcomes in `DEVELOPMENT_PLAN.md` as you execute these scenarios:

- Input construction (filesystem + auto-perf): load workflow, validate input,
  export JSON, and confirm namespace refs resolve.
- Input construction (git + URL): discover workflow, parse schema, validate and
  export inputs.
- Result analysis: load JSON/YAML/log results, parse and analyze, extract metrics,
  compare runs, and generate suggestions.
- Transport (stdio): initialize, tools/resources list, ping, and notification
  handling.
- Transport (HTTP/SSE): session binding, initialize, ping, and authorization
  checks.

### Manual validation

These steps are required before completing Phase 2 exit criteria:

1. Start local mode:
   `./server/arcaflow-mcp --mode local`
2. Connect with an MCP client (Claude Desktop/Gemini/Cursor).
3. Verify `initialize` response includes server capabilities.
4. Verify `tools/list` returns an empty list.
5. Send an invalid request and confirm MCP error response.
6. Confirm logs show connection lifecycle events.

### Phase 5 manual MCP client validation

Use an official MCP client (Claude Desktop or equivalent) and record results in
`DEVELOPMENT_PLAN.md`:

1. Start local mode: `./server/arcaflow-mcp --mode local`
2. Connect the client to the local MCP server.
3. Verify `initialize` response contains capabilities.
4. Verify `tools/list` and `resources/list` return expected entries.
5. Invoke a simple tool (e.g. `ping`) and confirm response shape.
6. Read a workflow resource and confirm schema payloads are returned.
7. Record any deviations or issues in the Phase 5 checklist.
