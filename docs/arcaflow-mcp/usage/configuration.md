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
  token_store_path: "/var/lib/arcaflow-mcp/tokens.json"
rate_limit:
  enabled: true
  requests_per_minute: 60
  window_seconds: 60
  backoff_enabled: true
  backoff_base_seconds: 1
  backoff_max_seconds: 60
tenancy:
  workspace_root: "/var/lib/arcaflow-mcp/tenants"
  tenant_store_path: "/var/lib/arcaflow-mcp/tenants.json"
  max_workspace_bytes: 0
  max_request_count: 0
  max_concurrent_requests: 10
  max_sessions: 4
audit:
  store_path: "/var/lib/arcaflow-mcp/audit.json"
  retention_days: 30
usage:
  store_path: "/var/lib/arcaflow-mcp/usage.json"
analysis:
  analysis_http_url: "http://127.0.0.1:8081"
```

Tenant IDs are required when minting tokens via
`POST /admin/tenants/{tenant_id}/tokens`, and the tenant must already exist
(create it with `POST /admin/tenants`). Tenant IDs are not auto-generated.
Tenant IDs must match `[A-Za-z0-9_.-]` and be 1-128 characters.

### Environment overrides

- `ARCAFLOW_MCP_MODE` (`local` or `server`)
- `ARCAFLOW_MCP_ADDRESS` (host:port)
- `ARCAFLOW_MCP_LOG_LEVEL` (`debug`, `info`, `warn`, `error`)
- `ARCAFLOW_MCP_ADMIN_TOKEN` (server-mode admin bearer token)
- `ARCAFLOW_MCP_TOKEN_STORE_PATH` (server-mode token store file path)
- `ARCAFLOW_MCP_RATE_LIMIT_ENABLED` (`true`/`false`)
- `ARCAFLOW_MCP_RATE_LIMIT_RPM` (requests per minute)
- `ARCAFLOW_MCP_RATE_LIMIT_WINDOW_SECONDS` (window size in seconds)
- `ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_ENABLED` (`true`/`false`)
- `ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_BASE_SECONDS` (minimum backoff in seconds)
- `ARCAFLOW_MCP_RATE_LIMIT_BACKOFF_MAX_SECONDS` (max backoff in seconds)
- `ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT` (root for tenant workspaces)
- `ARCAFLOW_MCP_TENANT_STORE_PATH` (tenant store file path)
- `ARCAFLOW_MCP_TENANT_MAX_WORKSPACE_BYTES` (max workspace bytes, 0 disables)
- `ARCAFLOW_MCP_TENANT_MAX_REQUESTS` (max requests per tenant, 0 disables)
- `ARCAFLOW_MCP_TENANT_MAX_CONCURRENT` (per-tenant concurrent requests)
- `ARCAFLOW_MCP_TENANT_MAX_SESSIONS` (per-tenant SSE sessions)
- `ARCAFLOW_MCP_AUDIT_STORE_PATH` (audit store file path)
- `ARCAFLOW_MCP_AUDIT_RETENTION_DAYS` (audit retention in days)
- `ARCAFLOW_MCP_USAGE_STORE_PATH` (usage store file path)
- `ARCAFLOW_MCP_ANALYSIS_HTTP_URL` (analysis service HTTP base URL)

### Analysis service integration

When the analysis service is running with its HTTP endpoint enabled, configure
the MCP server to reach it by setting `analysis.analysis_http_url` in the YAML
config or `ARCAFLOW_MCP_ANALYSIS_HTTP_URL` as an environment override.

### Authentication notes

Server mode requires an admin token to mint tenant tokens. Local mode bypasses
authentication entirely. Server mode also requires `auth.token_store_path` and
`tenancy.tenant_store_path` so tenant tokens and tenant records persist across
restarts, plus `audit.store_path` and `usage.store_path` for audit and usage
retention. Rate limiting is applied per tenant in server mode with progressive
backoff. Tenancy settings control workspace isolation and limits.
