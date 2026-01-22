## Kubernetes deployment

This section will document Kubernetes deployment patterns and manifests.

### Persistence requirement

Server mode needs persistent storage to survive restarts. Tenant records and
tenant tokens require durable storage. Configure
`auth.token_store_path` and `tenancy.tenant_store_path` to point at
PersistentVolumeClaims so data survives restarts. Configure `audit.store_path`
and `usage.store_path` on PersistentVolumeClaims to persist audit and usage
records.
- Tenant workspaces (`tenancy.workspace_root`) must be on shared storage for
  clustered deployments so each pod sees the same workspace contents.

Analysis service persistence:

- The analysis service requires a writable volume for its SQLite database when
  running in local or single-node mode.
- For production, configure PostgreSQL and provide a persistent backend for
  historical analysis data.

Analysis service integration:

- Set `ARCAFLOW_MCP_ANALYSIS_HTTP_URL` to the analysis service HTTP base URL
  (for example, `http://analysis-service:8081`) so the MCP server can reach it.

### Clustered deployment requirements

Running multiple replicas requires additional coordination:

- Tenant records are persisted via `tenancy.tenant_store_path`. Use a shared,
  durable backend before scaling beyond a single replica.
- SSE sessions are stored in-memory per pod. Clients must reach the same pod for
  `/mcp/events` and subsequent `POST /mcp` requests; configure load balancing
  with session affinity or stickiness.
- Token persistence relies on `auth.token_store_path`. For multiple replicas,
  the token store must point at shared storage (ReadWriteMany PVC or an external
  token backend) so all pods see the same tokens.
- Rate limiting and concurrency limits are enforced per instance. For cluster
  wide enforcement, move these counters into shared storage.
- Audit records are stored via `audit.store_path`. For multiple replicas, use a
  shared backend or ReadWriteMany PVC so all pods see the same audit history.
- Usage statistics are stored via `usage.store_path`. For multiple replicas, use
  shared storage to keep counts consistent across pods.
