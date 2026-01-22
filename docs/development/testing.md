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

### Manual validation

These steps are required before completing Phase 2 exit criteria:

1. Start local mode:
   `./server/arcaflow-mcp --mode local`
2. Connect with an MCP client (Claude Desktop/Gemini/Cursor).
3. Verify `initialize` response includes server capabilities.
4. Verify `tools/list` returns an empty list.
5. Send an invalid request and confirm MCP error response.
6. Confirm logs show connection lifecycle events.
