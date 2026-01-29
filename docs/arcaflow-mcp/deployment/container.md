# Container Deployment

Deploy Arcaflow MCP Server in containers using Docker or Podman for reproducible, isolated environments.

## Table of Contents

1. [Overview](#overview)
2. [Container Images](#container-images)
3. [Quick Start](#quick-start)
4. [Persistence Requirements](#persistence-requirements)
5. [Environment Variables](#environment-variables)
6. [Docker Deployment](#docker-deployment)
7. [Podman Deployment](#podman-deployment)
8. [Docker Compose](#docker-compose)
9. [Podman Compose](#podman-compose)
10. [Health Checks](#health-checks)
11. [Troubleshooting](#troubleshooting)

---

## Overview

Containerization provides:

- **Reproducibility**: Consistent environment across deployments
- **Isolation**: Process and filesystem isolation
- **Portability**: Run on any container runtime
- **Scalability**: Easy horizontal scaling
- **Resource Control**: CPU/memory limits per container

**Container Runtimes Supported:**

- Docker
- Podman
- Kubernetes (see [kubernetes.md](kubernetes.md))

---

## Container Images

### Official Images

```bash
# Go MCP Server
quay.io/arcalot/arcaflow-mcp-server:<tag>

# Python Analysis Engine
quay.io/arcalot/arcaflow-mcp-analysis:<tag>
```

> **🚨 Pre-Release Container Tags (Before v0.1.0)**
>
> We're currently in active development. Container tags change with each commit.
>
> **Get the current tag:**
> ```bash
> export TAG=$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash)
> echo "Current tag: $TAG"
> # Example output: main-abc1234
> ```
>
> **After v0.1.0 release**, use stable tags:
> - `:latest` - Latest stable build
> - `:v1.0.0`, `:v1.0.1` - Specific version releases

### Image Tags

**Available Tag Types:**
- `latest`: Latest build from main branch (after v0.1.0)
- `v1.0.0`, `v1.0.1`, etc.: Specific releases (after v0.1.0)
- `main-<sha>`: Specific commit builds (7-char SHA) - current development

**For Production**: Use specific version tags (e.g., `v1.0.0`) once available. The `latest` tag tracks the main branch and may include breaking changes.

### Building from Source

**Go Server (build from repo root):**

```bash
# Build from repository root to access VERSION file
podman build -t arcaflow-mcp-server:local -f server/Containerfile .
# Or with Docker:
# docker build -t arcaflow-mcp-server:local -f server/Containerfile .
```

**Python Analysis Engine:**

```bash
# Build from repository root (context needs access to LICENSE/README)
podman build -t arcaflow-mcp-analysis:local -f analysis/Containerfile .
# Or with Docker:
# docker build -t arcaflow-mcp-analysis:local -f analysis/Containerfile .
```

**Containerfiles:**

- Go Server: [`server/Containerfile`](../../../server/Containerfile)
- Python Analysis Engine: [`analysis/Containerfile`](../../../analysis/Containerfile)

Both Containerfiles use multi-stage builds for minimal image size, run as non-root users, and include health checks.

### Finding Available Tags

Query Quay.io to discover available version tags:

```bash
# List all tags using skopeo
skopeo list-tags docker://quay.io/arcalot/arcaflow-mcp-server

# Or use Quay.io API directly
curl -s 'https://quay.io/api/v1/repository/arcalot/arcaflow-mcp-server/tag/?limit=100' | \
  jq -r '.tags[].name' | sort -V

# Browse visually: https://quay.io/repository/arcalot/arcaflow-mcp-server?tab=tags
```

**Using Specific Tags:**

```bash
# For latest build
podman pull quay.io/arcalot/arcaflow-mcp-server:latest

# For specific version (after v0.1.0)
podman pull quay.io/arcalot/arcaflow-mcp-server:v1.0.0

# For specific commit (if needed)
podman pull quay.io/arcalot/arcaflow-mcp-server:main-abc1234
```

Without cloning the repository:

```bash
# Fetch latest main branch SHA from GitHub
export TAG="main-$(curl -s https://api.github.com/repos/arcalot/arcaflow-mcp/commits/main | jq -r '.sha[:7]')"
echo "Using tag: $TAG"

# Pull images with this tag
podman pull quay.io/arcalot/arcaflow-mcp-server:${TAG}
podman pull quay.io/arcalot/arcaflow-mcp-analysis:${TAG}
```

**Browsing Available Tags:**

Visit the Quay.io repositories to see all available tags:
- Go Server: https://quay.io/repository/arcalot/arcaflow-mcp-server?tab=tags
- Python Engine: https://quay.io/repository/arcalot/arcaflow-mcp-analysis?tab=tags

**Stability:**

Development builds (`main-<sha>`) are:
- Built automatically on every commit to main
- Useful for testing latest features
- Not recommended for production
- Expire after 90 days

For production use, wait for official releases or build from a specific git tag.

---

## Quick Start

### Single Container (stdio mode)

```bash
# Run MCP server in stdio mode (for local testing)
docker run -i --rm \
  -v "$PWD/workflows:/workflows:ro" \
  quay.io/arcalot/arcaflow-mcp-server:latest \
  --stdio
```

### Multi-Container (server mode)

```bash
# Start analysis engine
docker run -d --name analysis \
  -p 8081:8081 \
  -v arcaflow-analysis-data:/var/lib/arcaflow-analysis \
  quay.io/arcalot/arcaflow-mcp-analysis:latest

# Start MCP server
docker run -d --name mcp-server \
  -p 8080:8080 \
  -e ARCAFLOW_MCP_ADMIN_TOKEN="your-secure-admin-token" \
  -e ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://analysis:8081" \
  -v arcaflow-mcp-data:/var/lib/arcaflow-mcp \
  -v "$PWD/workflows:/workflows:ro" \
  --link analysis:analysis \
  quay.io/arcalot/arcaflow-mcp-server:latest \
  --server
```

---

## Persistence Requirements

### Server Mode Data Persistence

Server mode requires persistent storage to survive container restarts.

**Required Persistent Volumes:**

1. **Tenant Store**: `/var/lib/arcaflow-mcp/tenants`
2. **Token Store**: `/var/lib/arcaflow-mcp/tokens`
3. **Audit Logs**: `/var/lib/arcaflow-mcp/audit`
4. **Usage Stats**: `/var/lib/arcaflow-mcp/usage`
5. **Tenant Workspaces**: `/var/lib/arcaflow-mcp/workspaces`

**Analysis Engine Persistence:**

1. **SQLite Database**: `/var/lib/arcaflow-analysis/arcaflow_analysis.db` (development)
2. **PostgreSQL**: External database (production)

### Volume Strategies

#### Named Volumes (Recommended)

```bash
# Create named volumes
docker volume create arcaflow-mcp-data
docker volume create arcaflow-analysis-data

# Use in containers
docker run -v arcaflow-mcp-data:/var/lib/arcaflow-mcp ...
docker run -v arcaflow-analysis-data:/var/lib/arcaflow-analysis ...
```

#### Bind Mounts

```bash
# Create host directories
mkdir -p /opt/arcaflow-mcp/data
mkdir -p /opt/arcaflow-analysis/data

# Set permissions
chmod 755 /opt/arcaflow-mcp/data
chmod 755 /opt/arcaflow-analysis/data

# Use in containers
docker run -v /opt/arcaflow-mcp/data:/var/lib/arcaflow-mcp ...
docker run -v /opt/arcaflow-analysis/data:/var/lib/arcaflow-analysis ...
```

---

## Environment Variables

### Go MCP Server

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ARCAFLOW_MCP_ADMIN_TOKEN` | Admin authentication token (for tenant management via `/admin/*` APIs, not for tenant MCP access) | - | Yes (server mode) |
| `ARCAFLOW_MCP_ANALYSIS_HTTP_URL` | Analysis engine URL | `http://localhost:8081` | No |
| `ARCAFLOW_MCP_TENANT_STORE_PATH` | Tenant store path | `/var/lib/arcaflow-mcp/tenants` | No |
| `ARCAFLOW_MCP_TOKEN_STORE_PATH` | Token store path | `/var/lib/arcaflow-mcp/tokens` | No |
| `ARCAFLOW_MCP_AUDIT_STORE_PATH` | Audit log path | `/var/lib/arcaflow-mcp/audit` | No |
| `ARCAFLOW_MCP_USAGE_STORE_PATH` | Usage stats path | `/var/lib/arcaflow-mcp/usage` | No |
| `ARCAFLOW_MCP_WORKSPACE_ROOT` | Tenant workspaces root | `/var/lib/arcaflow-mcp/workspaces` | No |
| `ARCAFLOW_MCP_LISTEN_ADDR` | HTTP listen address | `:8080` | No |

**Note:** Tenant tokens (for MCP operations) are created via the admin API using
`ARCAFLOW_MCP_ADMIN_TOKEN`. See [Post-Deployment Setup](#post-deployment-setup).

### Python Analysis Engine

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ANALYSIS_PORT` | HTTP server port | `8081` | No |
| `ANALYSIS_HOST` | HTTP server bind address | `0.0.0.0` | No |
| `DATABASE_URL` | Database connection string | `sqlite:///./arcaflow_analysis.db` | No |
| `LOG_LEVEL` | Logging level | `INFO` | No |

---

## Docker Deployment

### Standalone MCP Server (stdio mode)

```bash
docker run -i --rm \
  -v "$PWD/workflows:/workflows:ro" \
  quay.io/arcalot/arcaflow-mcp-server:latest \
  --stdio < input.jsonrpc
```

### MCP Server with Analysis Engine (server mode)

```bash
# Create network
docker network create arcaflow

# Start analysis engine
docker run -d \
  --name arcaflow-analysis \
  --network arcaflow \
  -v arcaflow-analysis-data:/var/lib/arcaflow-analysis \
  quay.io/arcalot/arcaflow-mcp-analysis:latest

# Start MCP server
docker run -d \
  --name arcaflow-mcp-server \
  --network arcaflow \
  -p 8080:8080 \
  -e ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)" \
  -e ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://arcaflow-analysis:8081" \
  -v arcaflow-mcp-data:/var/lib/arcaflow-mcp \
  -v "$PWD/workflows:/workflows:ro" \
  quay.io/arcalot/arcaflow-mcp-server:latest \
  --server
```

### With Custom Configuration

```bash
# Create config file
cat > config.yaml <<EOF
server:
  enabled: true
  listen_addr: ":8080"

auth:
  admin_token: "${ARCAFLOW_MCP_ADMIN_TOKEN}"

analysis:
  http_url: "http://arcaflow-analysis:8081"

tenancy:
  workspace_root: "/var/lib/arcaflow-mcp/workspaces"
  tenant_store_path: "/var/lib/arcaflow-mcp/tenants"

audit:
  store_path: "/var/lib/arcaflow-mcp/audit"
EOF

# Run with config
docker run -d \
  --name arcaflow-mcp-server \
  -p 8080:8080 \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  -v arcaflow-mcp-data:/var/lib/arcaflow-mcp \
  quay.io/arcalot/arcaflow-mcp-server:latest \
  --server --config /app/config.yaml
```

---

## Podman Deployment

Podman is a daemonless container engine that runs rootless containers.

### Rootless Deployment

```bash
# Create pod
podman pod create --name arcaflow -p 8080:8080

# Start analysis engine in pod
podman run -d \
  --name arcaflow-analysis \
  --pod arcaflow \
  -v arcaflow-analysis-data:/var/lib/arcaflow-analysis:Z \
  quay.io/arcalot/arcaflow-mcp-analysis:latest

# Start MCP server in pod
podman run -d \
  --name arcaflow-mcp-server \
  --pod arcaflow \
  -e ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)" \
  -e ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081" \
  -v arcaflow-mcp-data:/var/lib/arcaflow-mcp:Z \
  -v "$PWD/workflows:/workflows:ro,Z" \
  quay.io/arcalot/arcaflow-mcp-server:latest \
  --server
```

**Note**: `:Z` suffix on volumes enables SELinux labeling for rootless containers.

---

## Docker Compose

### Using the Repository Compose File

The repository includes a production-ready compose file at `deploy/container/compose.yml` that deploys both components with proper configuration.

**Quick Start:**

```bash
# Method 1: From repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp/deploy/container

# Method 2: Download directly
curl -O https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/deploy/container/compose.yml

# Set admin token (REQUIRED)
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"

# Start services
docker compose up -d

# View logs
docker compose logs -f

# Stop services
docker compose down
```

**Compose File Features:**

- ✅ Both components (Go MCP Server + Python Analysis Engine)
- ✅ Health checks for both services
- ✅ Persistent volumes for data
- ✅ Proper networking between components
- ✅ Environment variable configuration
- ✅ Optional workflow directory mounting
- ✅ Support for development tags (TAG environment variable)

**Environment Variables:**

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `ARCAFLOW_MCP_ADMIN_TOKEN` | Admin token | **Yes** | - |
| `TAG` | Image tag | No | `latest` |
| `IMAGE_REPO` | Registry | No | `quay.io/arcalot` |
| `LOG_LEVEL` | Logging level | No | `INFO` |
| `WORKFLOW_DIR` | Workflow directory | No | `./workflows` |

**Using Specific Version Tags:**

```bash
# Use latest (default)
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
docker compose up -d

# Or use specific version tag
export TAG="v1.0.0"
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
docker compose up -d
```

**See Also:** [deploy/container/README.md](../../../deploy/container/README.md) for advanced configuration and troubleshooting.

---

## Post-Deployment Setup

After containers are running, you must create tenants for your users. The
containers alone do not provide access - tenants and tokens are required.

### Quick Start: Create Your First Tenant

**1. Verify deployment is healthy:**

```bash
# Check MCP server
curl http://localhost:8080/health
# Expected: {"status":"healthy","version":"..."}

# Check analysis engine
curl http://localhost:8081/health
# Expected: {"status":"healthy"}
```

**2. Create your first tenant:**

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<'EOF'
{
  "tenant_id": "my-team",
  "display_name": "My Team",
  "metadata": {
    "department": "engineering",
    "contact": "team@example.com"
  }
}
EOF
```

**Response:**
```json
{
  "tenant_id": "my-team",
  "display_name": "My Team",
  "metadata": {...},
  "created_at": "2026-01-29T12:00:00Z"
}
```

**3. Create tenant token (users will use this):**

```bash
curl -X POST http://localhost:8080/admin/tenants/my-team/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | tee tenant-token.json
```

**Response (save this token securely):**
```json
{
  "token": "tnt_a1b2c3d4e5f6g7h8...",
  "tenant_id": "my-team",
  "created_at": "2026-01-29T12:05:00Z"
}
```

**4. Distribute token to users:**

Users configure their AI clients with the tenant token. Example for Claude Desktop:

```json
{
  "mcpServers": {
    "arcaflow": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "--network", "host",
        "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081",
        "quay.io/arcalot/arcaflow-mcp-server:latest",
        "--mode", "client",
        "--server-url", "http://localhost:8080",
        "--token", "tnt_a1b2c3d4e5f6g7h8..."
      ]
    }
  }
}
```

### Next Steps

**For complete tenant management:**
- 📖 [Authentication Guide](authentication.md) - Token management, rotation, security
- 📖 [Multi-Tenancy Concepts](../concepts/multi-tenancy.md) - Workspace isolation, quotas, usage tracking
- 📖 [Server Mode Usage](../usage/server-mode.md) - Admin API reference

**For production deployments:**
- Configure TLS: [TLS Configuration](tls.md)
- Set up monitoring: Check audit logs and usage metrics
- Establish token rotation policy
- Configure resource quotas per tenant

---

## Podman Compose

```bash
# Install podman-compose
pip install podman-compose

# Use same compose.yml as Docker
podman-compose -f deploy/container/compose.yml up -d

# Or use Podman pods
podman play kube deploy/container/pod.yaml
```

---

## Health Checks

### MCP Server Health Check

```bash
# Docker
docker exec arcaflow-mcp-server wget -qO- http://localhost:8080/health

# Podman
podman exec arcaflow-mcp-server wget -qO- http://localhost:8080/health
```

**Expected Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### Analysis Engine Health Check

```bash
# Docker
docker exec arcaflow-analysis curl -f http://localhost:8081/health

# Podman
podman exec arcaflow-analysis curl -f http://localhost:8081/health
```

---

## Troubleshooting

### Container Won't Start

**Check logs:**

```bash
docker logs arcaflow-mcp-server
podman logs arcaflow-mcp-server
```

**Common Issues:**

- Missing `ARCAFLOW_MCP_ADMIN_TOKEN`
- Permission denied on volume mounts
- Port already in use

### Permission Denied Errors

**Solution 1: Fix volume permissions**

```bash
# Create host directory with correct permissions
mkdir -p /opt/arcaflow-mcp/data
chmod 755 /opt/arcaflow-mcp/data

# Run container with correct user
docker run --user $(id -u):$(id -g) ...
```

**Solution 2: Use named volumes**

```bash
# Named volumes handle permissions automatically
docker volume create arcaflow-mcp-data
docker run -v arcaflow-mcp-data:/var/lib/arcaflow-mcp ...
```

### Analysis Engine Not Reachable

**Check network connectivity:**

```bash
# Docker
docker network inspect bridge

# Podman
podman network inspect podman
```

**Verify URL:**

```bash
# From MCP server container
docker exec arcaflow-mcp-server wget -qO- http://analysis:8081/health
```

### Data Not Persisting

**Verify volumes:**

```bash
# Docker
docker volume ls
docker volume inspect arcaflow-mcp-data

# Podman
podman volume ls
podman volume inspect arcaflow-mcp-data
```

**Check mount points:**

```bash
docker inspect arcaflow-mcp-server | grep -A 10 "Mounts"
```

---

## Related Documentation

- [Kubernetes Deployment](kubernetes.md) - Deploy on Kubernetes
- [Authentication](authentication.md) - Configure authentication
- [TLS Configuration](tls.md) - Enable HTTPS

---

*For local development, see [Development Setup](../../development/setup.md).*
