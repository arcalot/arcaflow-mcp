## Container deployment

This section will document Podman and Docker container usage.

### Persistence requirement

Server mode needs persistence to survive restarts. Tenant records and tenant
tokens require durable storage. Configure `auth.token_store_path` and
`tenancy.tenant_store_path` to point at volume mounts so tenant data survives
restarts. If running multiple containers behind a load balancer, the tenant and
token stores must use shared storage so data remains consistent across
instances. Configure `audit.store_path` on a volume mount to persist audit
records. Plan to mount persistent storage for usage statistics once that backend
is implemented.
Tenant workspaces (`tenancy.workspace_root`) must be on shared storage when
running multiple containers behind a load balancer.

### Local compose

Use Podman compose for local development:

- `podman compose -f deploy/container/compose.yml up`
