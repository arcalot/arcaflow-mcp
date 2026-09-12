## Server mode

Server mode exposes MCP over HTTP/SSE for multi-tenant deployments. The HTTP
POST endpoint is wired to the JSON-RPC handler; SSE streaming provides session
binding and relays responses when a session header is present.

### Quick Start

**Minimal server setup (development/testing):**

```bash
# 1. Set data directory location (change this path if needed)
export DATA_DIR="./data"

# 2. Create data directory
mkdir -p "$DATA_DIR"

# 3. Generate admin token for testing
# IMPORTANT: For production, use a cryptographically secure token (see notes below)
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-admin-token-$(date +%s)"

# 4. Configure data storage paths (all use $DATA_DIR)
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# 5. Start Python analysis engine (required for result analysis features)
cd analysis
poetry install
poetry run python -m arcaflow_analysis.server.http_server &
ANALYSIS_PID=$!
cd ..

# Wait for analysis engine to start
sleep 2

# 6. Start Go MCP server (verify you're in repository root: ls server/arcaflow-mcp)
export ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081"
./server/arcaflow-mcp --mode server --address :8080 &
SERVER_PID=$!

# Wait for server to start
sleep 2

# 7. In another terminal, initialize MCP session (part 1 - handshake request)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {},
    "clientInfo": {
      "name": "test-client",
      "version": "1.0"
    }
  }
}
EOF

# Expected response: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25",...}}

# 8. Complete initialization (part 2 - notification, no id)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "method": "initialized"
}
EOF

# 9. Now make tool calls (example: list available tools)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
EOF
```

**Important Notes:**
- **Two Components Required**: Server mode requires both the Go MCP server AND the Python analysis engine
  - Go server handles protocol, input construction
  - Python engine handles result analysis, optimization suggestions
  - They communicate via HTTP (default: `localhost:8081`)
- **Authentication**: Server mode requires `ARCAFLOW_MCP_ADMIN_TOKEN` to be set
- **Data storage**: By default, the server uses `/var/lib/arcaflow-mcp` (requires root). The example above overrides this to use `./data/` in the current directory
- **Token security**: For development/testing, the timestamp-based token above is acceptable. For production, generate a secure token:
  ```bash
  # Linux/macOS: Generate secure random token
  export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -hex 32)"
  
  # Or use uuidgen
  export ARCAFLOW_MCP_ADMIN_TOKEN="$(uuidgen)"
  ```
- **MCP Protocol State**: The Quick Start examples above establish MCP protocol state per the specification (initialize → initialized → ready). However, each separate curl command creates a new connection without SSE session binding.
- **Session Binding**: For AI clients or multi-step workflows, use SSE session binding (see [Session Binding](#session-binding) below) to maintain state across multiple requests.
- **Production deployments**: See [Multi-Tenancy Concepts](../concepts/multi-tenancy.md), [Container Deployment](../deployment/container.md), and [Kubernetes Deployment](../deployment/kubernetes.md)

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

**When is session binding needed?**
- **AI Clients**: Claude Desktop, Cursor, and other MCP clients automatically use SSE session binding to maintain conversation state
- **Multi-step workflows**: When you need to maintain state across multiple tool calls (load workflow → build inputs → validate → export)
- **Not needed for**: Single operations, admin endpoints, or independent tool calls

**How session binding works:**

Server mode uses an `Mcp-Session-Id` header to bind HTTP POST requests to an SSE stream:

1. Connect to `GET /mcp/events` to open the SSE stream
2. Read the `Mcp-Session-Id` response header (also sent as an SSE `session` event)
3. Include `Mcp-Session-Id` header in all subsequent `POST /mcp` requests

If an SSE session exists for the same tenant, `POST /mcp` requests without the session header are rejected with `400 Bad Request`.

**Example with session binding (bash):**

```bash
# Terminal 1: Open SSE stream and capture session ID
curl -N -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  http://localhost:8080/mcp/events 2>&1 | tee >(grep -m1 "Mcp-Session-Id" | cut -d: -f2 | tr -d ' ' > /tmp/session-id) &

# Wait for session ID
sleep 1
SESSION_ID=$(cat /tmp/session-id)
echo "Session ID: $SESSION_ID"

# Terminal 2: Make requests with session binding
# Initialize
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
EOF

# Complete initialization
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","method":"initialized"}
EOF

# Now all subsequent requests maintain state
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF
```

**Note:** For simple testing and single operations, SSE session binding is optional. The Quick Start examples above work without it.

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

**Create tenant:**

```bash
curl -X POST http://127.0.0.1:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<'EOF'
{
  "tenant_id": "tenant-a",
  "display_name": "Tenant A",
  "metadata": {
    "team": "platform"
  }
}
EOF
```

**Create tenant token** (tenant must already exist):

```bash
curl -X POST http://127.0.0.1:8080/admin/tenants/tenant-a/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<'EOF'
{}
EOF
```

**List tenant tokens:**

```bash
curl http://127.0.0.1:8080/admin/tenants/tenant-a/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

**Get tenant usage:**

```bash
curl http://127.0.0.1:8080/admin/usage/tenants/tenant-a \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

Usage responses include `workspace_bytes` for current workspace size.

**Query audit logs:**

```bash
curl "http://127.0.0.1:8080/admin/audit?tenant_id=tenant-a&action=mcp_request&limit=50" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

**Tenant MCP request example** (using tenant token):

```bash
# Replace <tenant-token> with actual token from token creation above
curl -X POST http://127.0.0.1:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <tenant-token>" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ping"
}
EOF
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
