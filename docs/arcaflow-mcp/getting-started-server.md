# Getting Started: Server Mode (Containers)

Deploy Arcaflow MCP as a shared service for teams with authentication,
multi-tenancy, and workspace isolation. Estimated time: **~10 minutes**.

Server mode runs both components as long-lived services behind an HTTP API
with bearer token authentication. Each team or user group gets an isolated
workspace called a "tenant."

---

## Prerequisites

- **Docker with Docker Compose**, or **Podman with podman-compose**
- **openssl** (for generating secure tokens)
- **curl** and **jq** (for verification and API calls)

---

## Step 1: Get the Compose File

The compose file deploys both components with networking, health checks,
and persistent volumes.

```bash
# Option A: Clone the repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp/deploy/container

# Option B: Download the compose file directly
mkdir arcaflow-mcp && cd arcaflow-mcp
curl -O https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/deploy/container/compose.yml
```

> [!NOTE]
> **Pre-release (before v0.1.0):** You may need to set a specific image tag.
> If you cloned the repo: `export TAG=$(./scripts/get-container-tag.sh)`
> Otherwise the compose file defaults to `:latest`.

---

## Step 2: Generate an Admin Token

The admin token is used to create tenants and manage access. Generate a
secure random token:

```bash
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
echo "Admin token: $ARCAFLOW_MCP_ADMIN_TOKEN"
```

> [!WARNING]
> Save this token securely. You will need it for all administrative operations
> (creating tenants, issuing user tokens). It cannot be recovered once lost.

---

## Step 3: Start Both Services

```bash
docker compose up -d
```

Or with Podman:

```bash
podman-compose up -d
```

This starts:
- **Analysis Engine** on port 8081 (result analysis and optimization)
- **MCP Server** on port 8080 (MCP protocol, workflow operations, authentication)

---

## Step 4: Verify Services Are Running

```bash
# Check container status
docker compose ps

# Check analysis engine health
curl http://localhost:8081/healthz
# Expected: {"status":"healthy"}

# Check MCP server health
curl http://localhost:8080/healthz
# Expected: {"status":"healthy","version":"..."}
```

Both endpoints should return healthy status before proceeding.

> [!TIP]
> If a service isn't healthy, check the logs:
> ```bash
> docker compose logs analysis
> docker compose logs mcp-server
> ```

---

## Step 5: Create Your First Tenant

Create a tenant workspace for your team:

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"my-team","display_name":"My Team"}'
```

Generate a token for users to access this tenant:

```bash
curl -s -X POST http://localhost:8080/admin/tenants/my-team/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.token'
```

> [!IMPORTANT]
> Save the token output. Users will need it to connect their MCP clients
> to the server.

---

## Try It Out

Configure your MCP client to connect to the server using the tenant token
from Step 5. See [Server Mode Usage](usage/server-mode.md) for client
configuration details, including SSE session binding.

Once connected, verify the setup by asking your AI agent:

1. *"Can you see the Arcaflow MCP tools? List them."*
2. *"List the available workflows"*
3. *"Show me the input template for one of the workflows"*

---

## What's Next

- [Server Mode Usage](usage/server-mode.md) -- client configuration, SSE sessions, authentication details
- [Authentication Guide](deployment/authentication.md) -- managing tenants and tokens
- [Multi-Tenancy Concepts](concepts/multi-tenancy.md) -- workspace isolation and quotas
- [Container Deployment](deployment/container.md) -- production hardening, TLS, resource limits
- [Kubernetes Deployment](deployment/kubernetes.md) -- deploying on Kubernetes
- [Troubleshooting](troubleshooting.md) -- common issues and solutions
