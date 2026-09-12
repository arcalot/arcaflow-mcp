## Persistence inventory

Arcaflow MCP has a small set of server-mode state that must survive restarts.
This inventory is the canonical list and should be updated whenever new
server-mode state is introduced.

### Persistent state (required)

- **Tenant records**: stored via `tenancy.tenant_store_path`.
- **Tenant tokens**: stored via `auth.token_store_path`.
- **Audit records**: stored via `audit.store_path` with retention controls.
- **Usage statistics**: stored via `usage.store_path`.
- **Tenant workspaces**: stored under `tenancy.workspace_root`.

### Non-persistent state (runtime only)

- **SSE sessions**: held in memory per server instance.
- **Rate limit counters**: enforced per instance.
- **Concurrency limit slots**: enforced per instance.

### Cluster considerations

For multi-replica deployments, all persistent state paths must point at shared
storage (for example, ReadWriteMany PVCs or external services). Runtime-only
state remains per instance and must be mitigated at the load balancer level
(session affinity for SSE, single replica if strict global rate limiting is
required).

### Backup and migration guidance

All file-backed stores use JSON to simplify backup and restore. When adding new
fields to persisted records, ensure defaults are applied on load so existing
stores remain compatible.
