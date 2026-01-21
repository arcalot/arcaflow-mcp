## Configuration

Configuration can be supplied via a YAML file and overridden by environment
variables.

### YAML example

```
mode: server
address: 127.0.0.1:8080
logging:
  level: info
auth:
  admin_token: "replace-with-secure-token"
rate_limit:
  enabled: true
  requests_per_minute: 60
  window_seconds: 60
  backoff_enabled: true
  backoff_base_seconds: 1
  backoff_max_seconds: 60
tenancy:
  workspace_root: "/tmp/arcaflow-mcp/tenants"
  max_concurrent_requests: 10
  max_sessions: 4
```

Tenant IDs are required when minting tokens via `POST /admin/tokens`. They are
not auto-generated.

### Environment overrides

- `ARCAFLOW_MCP_MODE` (`local` or `server`)
- `ARCAFLOW_MCP_ADDRESS` (host:port)
- `ARCAFLOW_MCP_LOG_LEVEL` (`debug`, `info`, `warn`, `error`)
- `ARCAFLOW_MCP_ADMIN_TOKEN` (server-mode admin bearer token)
- `ARCAFLOW_MCP_RATE_LIMIT_ENABLED` (`true`/`false`)
- `ARCAFLOW_MCP_RATE_LIMIT_RPM` (requests per minute)
- `ARCAFLOW_MCP_RATE_LIMIT_WINDOW_SECONDS` (window size in seconds)
- `ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_ENABLED` (`true`/`false`)
- `ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_BASE_SECONDS` (minimum backoff in seconds)
- `ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_MAX_SECONDS` (max backoff in seconds)
- `ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT` (root for tenant workspaces)
- `ARCAFLOW_MCP_TENANT_MAX_CONCURRENT` (per-tenant concurrent requests)
- `ARCAFLOW_MCP_TENANT_MAX_SESSIONS` (per-tenant SSE sessions)

### Authentication notes

Server mode requires an admin token to mint tenant tokens. Local mode bypasses
authentication entirely. Rate limiting is applied per tenant in server mode with
progressive backoff. Tenancy settings control workspace isolation and limits.
