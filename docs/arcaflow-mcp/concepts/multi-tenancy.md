# Multi-Tenancy

**Secure Multi-User Deployment with Workspace Isolation**

Server mode provides multi-tenancy with isolated workspaces, independent authentication, and per-tenant resource controls. Each tenant operates in a secure, isolated environment with dedicated storage and quotas.

---

## Overview

Multi-tenancy enables:
- **Multiple teams or users** sharing a single MCP server deployment
- **Workspace isolation** preventing data leakage between tenants
- **Independent authentication** with per-tenant token management
- **Resource quotas** for fair resource allocation
- **Usage tracking** for billing, monitoring, and capacity planning
- **Audit logging** for compliance and security

---

## Tenant Model

### Tenant Identity

Each tenant has:
- **Tenant ID**: Unique identifier (e.g., `team-platform`, `user-alice`)
  - Must match `[A-Za-z0-9_.-]` pattern
  - 1-128 characters in length
  - Immutable after creation
- **Display Name**: Human-readable name (e.g., "Platform Team")
- **Metadata**: Key-value pairs for organizational info (team, cost center, etc.)

### Tenant Workspace

Each tenant gets an isolated workspace directory:

```
$ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT/
├── tenant-a/
│   ├── workflows/          # Discovered workflows
│   ├── inputs/             # Built workflow inputs
│   ├── results/            # Loaded analysis results
│   └── .cache/             # Tenant-specific cache
├── tenant-b/
│   └── ...
└── tenant-c/
    └── ...
```

**Isolation Guarantees:**
- Tenants cannot access each other's workspaces
- Filesystem paths are restricted to tenant directory
- Workflow loading scoped to tenant workspace
- Result storage isolated per tenant

---

## Authentication

### Token Types

**Admin Token:**
- Configured via `ARCAFLOW_MCP_ADMIN_TOKEN` environment variable
- Used to manage tenants and provision tenant tokens
- Has access to admin API endpoints
- Should be kept secret and rotated regularly

**Tenant Tokens:**
- Created by admin for specific tenants
- Scoped to single tenant (cannot access other tenants)
- Used for MCP operations (workflow loading, result analysis)
- Can be listed, created, and revoked via admin API

### Token Management

**Create Tenant:**
```bash
curl -X POST http://server:8080/admin/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "tenant_id": "team-qa",
  "display_name": "QA Team",
  "metadata": {"cost_center": "engineering"}
}
EOF
```

**Create Tenant Token:**
```bash
curl -X POST http://server:8080/admin/tenants/team-qa/tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{}' 
```

**Response includes the token** (store securely):
```json
{
  "token": "tnt_a1b2c3d4e5f6...",
  "tenant_id": "team-qa",
  "created_at": "2026-01-28T12:00:00Z"
}
```

**Revoke Token:**
```bash
curl -X DELETE \
  http://server:8080/admin/tenants/team-qa/tokens/tnt_a1b2c3d4e5f6... \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

See [Authentication Guide](../deployment/authentication.md) for complete token management.

---

## Resource Quotas

### Per-Tenant Limits

Administrators can configure per-tenant quotas:

**Workspace Storage:**
- `max_workspace_bytes` - Maximum disk space for tenant workspace
- Default: 0 (unlimited)
- Enforced on workflow loading and result storage

**Request Limits:**
- `max_request_count` - Total requests before tenant is locked
- Default: 0 (unlimited)
- Useful for trial accounts or cost control

**Concurrent Operations:**
- `max_concurrent_requests` - Simultaneous requests per tenant
- Default: 10
- Prevents single tenant monopolizing server resources

**Sessions:**
- `max_sessions` - Maximum SSE sessions per tenant
- Default: 4
- Limits number of concurrent AI clients per tenant

### Quota Enforcement

**When quota exceeded:**
1. Request is rejected with `429 Too Many Requests` or `403 Forbidden`
2. Response includes quota information
3. Audit log records the violation
4. Admin can review usage metrics and adjust quotas

**Example quota error response:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "workspace quota exceeded",
    "data": {
      "current_bytes": 1073741824,
      "max_bytes": 1073741824,
      "tenant_id": "team-qa"
    }
  }
}
```

---

## Usage Tracking

### Metrics Collected

Per tenant, the server tracks:
- **Request count**: Total MCP requests
- **Tool calls**: Count by tool name
- **Workspace size**: Current disk usage in bytes
- **Session count**: Active SSE sessions
- **First seen**: Tenant creation timestamp
- **Last active**: Most recent request timestamp

### Viewing Usage

**Get tenant usage:**
```bash
curl http://server:8080/admin/usage/tenants/team-qa \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Response:**
```json
{
  "tenant_id": "team-qa",
  "request_count": 1523,
  "tool_calls": {
    "workflow_load": 89,
    "workflow_input_build": 234,
    "workflow_results_analyze": 156
  },
  "workspace_bytes": 524288000,
  "session_count": 2,
  "first_seen": "2026-01-15T08:00:00Z",
  "last_active": "2026-01-28T14:30:00Z"
}
```

### Usage Reporting

**List all tenant usage:**
```bash
curl http://server:8080/admin/usage/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Use for:
- Capacity planning
- Cost allocation
- Identifying heavy users
- Optimizing resource allocation

---

## Audit Logging

### Audit Events

Server mode logs all tenant operations:
- **MCP requests**: Tool calls, resource reads, ping
- **SSE events**: Session creation, message delivery
- **Admin operations**: Tenant creation, token management
- **Authentication**: Login attempts, token validation
- **Errors**: Failed requests, quota violations

### Audit Log Format

Each log entry includes:
- Timestamp (RFC3339)
- Tenant ID
- Action (e.g., `mcp_request`, `token_created`)
- Outcome (`success`, `failure`)
- HTTP method and path
- Status code
- Additional context (error details, quotas, etc.)

### Querying Audit Logs

**Filter by tenant:**
```bash
curl "http://server:8080/admin/audit?tenant_id=team-qa&limit=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Filter by action:**
```bash
curl "http://server:8080/admin/audit?action=mcp_request&limit=50" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Time range:**
```bash
curl "http://server:8080/admin/audit?start=2026-01-27T00:00:00Z&end=2026-01-28T00:00:00Z" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Audit Retention

- Configurable retention period (`audit.retention_days`, default 30 days)
- Automatic cleanup of old entries
- Storage path: `$ARCAFLOW_MCP_AUDIT_STORE_PATH`

---

## Rate Limiting

### Per-Tenant Rate Limits

Prevents resource exhaustion and ensures fair access:

**Configuration:**
- `rate_limit.requests_per_minute` - Max requests per tenant per minute (default: 60)
- `rate_limit.window_seconds` - Rolling window size (default: 60)
- `rate_limit.backoff_enabled` - Progressive penalties for repeated violations
- `rate_limit.backoff_max_seconds` - Maximum backoff delay (default: 60)

### Rate Limit Headers

All responses include standard rate limit headers:
- `RateLimit-Limit`: Configured requests per minute
- `RateLimit-Remaining`: Requests remaining in current window
- `RateLimit-Reset`: Unix timestamp when limit resets

**Example:**
```
RateLimit-Limit: 60
RateLimit-Remaining: 45
RateLimit-Reset: 1706443200
```

### Backoff Behavior

On consecutive rate limit violations:
1. First violation: Immediate 429 response
2. Second violation: 429 + 1-second delay suggested
3. Third violation: 429 + 2-second delay
4. Continues doubling up to `backoff_max_seconds`
5. Resets after successful request within limit

---

## Security and Isolation

### Workspace Isolation

**Filesystem:**
- Tenants can only access their workspace directory
- Path traversal attacks prevented
- Symbolic links restricted
- No access to other tenant directories or system files

**Process:**
- All operations run as server process user
- No tenant-specific process isolation (use container namespaces if required)
- Resource limits enforced via quotas

### Network Isolation

**Not provided** - All tenants share:
- Server process and network connections
- Analysis engine (if deployed separately)
- System resources (CPU, memory)

For network isolation, deploy separate server instances or use:
- Kubernetes network policies
- Container isolation
- Separate infrastructure per tenant

### Data Leakage Prevention

**Controls:**
- Workflow loading restricted to tenant workspace
- Result analysis scoped to tenant data
- No cross-tenant queries supported
- Session state isolated per tenant
- Audit logs track all cross-boundary attempts

---

## Admin Operations

### Tenant Management

**Create Tenant:**
```bash
POST /admin/tenants
{
  "tenant_id": "new-team",
  "display_name": "New Team",
  "metadata": {"department": "engineering"}
}
```

**List Tenants:**
```bash
GET /admin/tenants
```

**Get Tenant Details:**
```bash
GET /admin/tenants/team-qa
```

**Update Tenant:**
```bash
PUT /admin/tenants/team-qa
{
  "display_name": "Updated Name",
  "metadata": {"department": "qa", "region": "us-west"}
}
```

**Delete Tenant:**
```bash
DELETE /admin/tenants/team-qa
```

**Warning:** Deleting a tenant also:
- Revokes all tenant tokens
- Deletes tenant workspace (if configured)
- Removes audit logs after retention period
- Cannot be undone

---

## Use Cases

### Team Collaboration

**Scenario:** QA team needs shared access to performance test workflows.

**Setup:**
1. Admin creates `team-qa` tenant
2. Admin creates tokens for each QA engineer
3. Each engineer configures their AI client with tenant token
4. Team shares workflows in server workspace
5. Results and optimizations shared across team

### Multi-Project Organization

**Scenario:** Organization has multiple projects, each with own workflows.

**Setup:**
1. Create tenant per project (`project-web`, `project-mobile`, `project-api`)
2. Each project team gets scoped tokens
3. Workspace isolation prevents cross-project access
4. Usage tracking shows resource consumption per project
5. Audit logs track activity per project

### Enterprise SaaS

**Scenario:** Provide Arcaflow workflow assistance as a service.

**Setup:**
1. Create tenant per customer organization
2. Enforce quotas to control costs
3. Track usage for billing
4. Audit logs for compliance
5. Rate limiting for fair resource allocation

---

## Best Practices

### Tenant Naming

- **Use descriptive IDs**: `team-platform` not `t1`
- **Consistent naming**: Follow organizational convention
- **Consider hierarchy**: `dept-eng-team-qa` for clarity

### Token Management

- **Rotate tokens regularly**: Implement token rotation policy
- **Store tokens securely**: Use secrets management (Vault, etc.)
- **One token per user**: Don't share tokens across users
- **Revoke promptly**: Remove tokens when users leave team

### Quota Configuration

- **Start conservative**: Set reasonable initial quotas
- **Monitor usage**: Review metrics regularly
- **Adjust as needed**: Increase quotas based on actual usage
- **Alert on limits**: Monitor quota violations

### Audit and Compliance

- **Regular audit reviews**: Check logs for anomalies
- **Retention policy**: Balance compliance needs with storage costs
- **Export for archival**: Periodically export audit logs for long-term storage
- **Automated alerting**: Monitor for suspicious activity

---

## Troubleshooting

### Common Issues

**"Unauthorized" errors:**
- Verify token is valid and not revoked
- Check token matches tenant
- Ensure `Authorization: Bearer <token>` header is present

**"Workspace quota exceeded":**
- Check current usage with `GET /admin/usage/tenants/{id}`
- Clean up old workflow data in tenant workspace
- Request quota increase from admin

**"Too many sessions":**
- Close unused AI client connections
- Check `max_sessions` quota
- Request quota increase if legitimate need

**"Rate limit exceeded":**
- Slow down request rate
- Wait for rate limit window to reset (check `RateLimit-Reset` header)
- Implement exponential backoff in automation scripts

---

## Related Documentation

- **[Server Mode](../usage/server-mode.md)** - Server deployment and configuration
- **[Authentication](../deployment/authentication.md)** - Authentication setup
- **[Deployment Modes](deployment-modes.md)** - Local vs Server comparison
- **[Configuration](../usage/configuration.md)** - Complete configuration reference

---

[← Back to Documentation Index](../index.md)
