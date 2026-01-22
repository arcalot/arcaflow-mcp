## Container deployment

This section will document Podman and Docker container usage.

### Persistence requirement

Server mode needs persistence to survive restarts. Tenant records and tenant
tokens require durable storage. Configure `auth.token_store_path` and
`tenancy.tenant_store_path` to point at volume mounts so tenant data survives
restarts. If running multiple containers behind a load balancer, the tenant and
token stores must use shared storage so data remains consistent across
instances. Configure `audit.store_path` and `usage.store_path` on volume mounts
to persist audit and usage records.
Tenant workspaces (`tenancy.workspace_root`) must be on shared storage when
running multiple containers behind a load balancer.

Analysis service persistence:

- The Python analysis service requires SQLite for local persistence by default.
- Mount a writable volume for the SQLite database path in containers.
- For production, configure the analysis service to use PostgreSQL instead of
  SQLite.

Analysis service integration:

- Set `ARCAFLOW_MCP_ANALYSIS_HTTP_URL` to the analysis service HTTP base URL
  (for example, `http://analysis:8081`) so the MCP server can reach it.

### Local compose

Use Podman compose for local development:

- `podman compose -f deploy/container/compose.yml up`
