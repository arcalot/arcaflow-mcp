## Server mode

Server mode exposes MCP over HTTP/SSE for multi-tenant deployments. The HTTP
POST endpoint is wired to the JSON-RPC handler; SSE streaming provides session
binding and relays responses when a session header is present.

### Endpoints (partial)

- `POST /mcp` handles JSON-RPC client-to-server messages (including `ping`)
- `POST /mcp` returns `204 No Content` for notifications (no `id`)
- `GET /mcp/events` streams server-to-client SSE events (`session`, `message`)
- `GET /admin/tenants` lists tenant records (admin only)
- `POST /admin/tenants` creates tenant records (admin only)
- `GET /admin/tenants/{tenant_id}` fetches a tenant record (admin only)
- `PUT /admin/tenants/{tenant_id}` updates a tenant record (admin only)
- `DELETE /admin/tenants/{tenant_id}` deletes a tenant record (admin only)
- `GET /admin/tenants/{tenant_id}/tokens` lists tenant tokens (admin only)
- `POST /admin/tenants/{tenant_id}/tokens` creates tenant tokens (admin only)
- `DELETE /admin/tenants/{tenant_id}/tokens/{token}` revokes tenant tokens
  (admin only)
- `GET /admin/usage/tenants` lists tenant usage metrics (admin only)
- `GET /admin/usage/tenants/{tenant_id}` fetches tenant usage metrics (admin only)
- `GET /admin/audit` queries audit logs (admin only)
- `GET /healthz` returns a simple health response

### Session binding

Server mode uses an `Mcp-Session-Id` header to bind HTTP POST requests to an SSE
stream:

1. Connect to `GET /mcp/events` to open the SSE stream.
2. Read the `Mcp-Session-Id` response header (also sent as an SSE `session`
   event).
3. Include `Mcp-Session-Id` on `POST /mcp` requests.

If an SSE session exists for the same tenant, `POST /mcp` requests without the
session header are rejected with `400 Bad Request`.

### Authentication

Server mode requires bearer tokens for all MCP and SSE endpoints. The admin
token is configured via `ARCAFLOW_MCP_ADMIN_TOKEN` (or `auth.admin_token` in the
config file) and is used to mint tenant tokens. Tenant tokens are persisted to
the file defined by `ARCAFLOW_MCP_TOKEN_STORE_PATH` (or `auth.token_store_path`),
tenant records are persisted via `tenancy.tenant_store_path`, and usage
statistics are persisted via `usage.store_path`.

1. Create a tenant record using the admin endpoint.
2. Create a tenant token scoped to that tenant.
3. Provide `Authorization: Bearer <token>` on `POST /mcp` and `GET /mcp/events`.

Tenant admin example:

```
curl -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"tenant-a","display_name":"Tenant A","metadata":{"team":"platform"}}' \
  http://127.0.0.1:8080/admin/tenants
```

Tenant token example (tenant must already exist):

```
curl -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' \
  http://127.0.0.1:8080/admin/tenants/tenant-a/tokens
```

Tenant token list example:

```
curl -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  http://127.0.0.1:8080/admin/tenants/tenant-a/tokens
```

Tenant usage example:

```
curl -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  http://127.0.0.1:8080/admin/usage/tenants/tenant-a
```

Usage responses include `workspace_bytes` for current workspace size.

Audit query example:

```
curl -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  "http://127.0.0.1:8080/admin/audit?tenant_id=tenant-a&action=mcp_request&limit=50"
```

Tenant request example:

```
curl -H "Authorization: Bearer <tenant-token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"ping"}' \
  http://127.0.0.1:8080/mcp
```

### Rate limiting

Server mode applies per-tenant rate limiting with progressive backoff on
consecutive violations. Successful and rejected requests include standard rate
limit headers:

- `RateLimit-Limit`
- `RateLimit-Remaining`
- `RateLimit-Reset` (unix timestamp)

Requests exceeding the limit receive `429 Too Many Requests` plus a
`Retry-After` header that grows with repeated violations.

### Audit logging

Server mode emits structured audit logs for authenticated requests and security
events. Each audit entry includes `tenant_id`, `action`, `outcome`, `status`,
`method`, and `path`.

### Tenant isolation

Server mode creates a per-tenant workspace directory for isolation. Tenant
requests are subject to configurable concurrency and SSE session limits.

### Persistence requirement

Server mode requires persistent storage to survive restarts:

- Tenant records created via `/admin/tenants` are stored in the tenant store
  file configured by `tenancy.tenant_store_path`.
- Tenant tokens minted via `/admin/tenants/{tenant_id}/tokens` are stored in the
  token store file configured by `auth.token_store_path`.
- Audit records are stored in the file configured by `audit.store_path`, with
  optional retention via `audit.retention_days`.
- Usage statistics are stored in the file configured by `usage.store_path`.

### Example flow

Start the SSE stream (capture the `Mcp-Session-Id` header from the response):

```
curl -N -H "Authorization: Bearer <tenant-token>" \
  http://127.0.0.1:8080/mcp/events
```

Send a JSON-RPC request with the session header (response is returned in the
HTTP body and also emitted as an SSE `message` event):

```
curl -H "Content-Type: application/json" \
  -H "Authorization: Bearer <tenant-token>" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}' \
  http://127.0.0.1:8080/mcp
```

### Running server mode

```
./server/arcaflow-mcp --mode server --address 127.0.0.1:8080
```

### Deployment sizing

Recommended minimums for server mode:

- CPU: 2 vCPU
- Memory: 4 GB RAM
- Disk: 10 GB plus tenant workspace storage

These values provide headroom for concurrent tenants, SSE sessions, audit
logging, and workspace creation. Increase memory and disk as workspace usage and
tenant concurrency grow.

### Notes

- Authentication is required in server mode and bypassed in local mode.
- Tenant quotas apply when `tenancy.max_workspace_bytes` or
  `tenancy.max_request_count` are set. Workspace limits return `403 Forbidden`;
  request quotas return `429 Too Many Requests`.
- Use local stdio mode for full MCP capabilities until HTTP/SSE is complete.
