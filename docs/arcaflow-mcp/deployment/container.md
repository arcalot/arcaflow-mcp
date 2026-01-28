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
quay.io/arcalot/arcaflow-mcp-server:latest
quay.io/arcalot/arcaflow-mcp-server:v1.0.0

# Python Analysis Engine
quay.io/arcalot/arcaflow-mcp-analysis:latest
quay.io/arcalot/arcaflow-mcp-analysis:v1.0.0
```

### Image Tags

- `latest`: Latest stable release (not recommended for production)
- `v1.0.0`: Specific version (recommended for production)
- `main`: Latest main branch build (unstable, for testing only)

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
  ghcr.io/arcaflow-mcp-server:latest \
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
| `ARCAFLOW_MCP_ADMIN_TOKEN` | Admin authentication token | - | Yes (server mode) |
| `ARCAFLOW_MCP_ANALYSIS_HTTP_URL` | Analysis engine URL | `http://localhost:8081` | No |
| `ARCAFLOW_MCP_TENANT_STORE_PATH` | Tenant store path | `/var/lib/arcaflow-mcp/tenants` | No |
| `ARCAFLOW_MCP_TOKEN_STORE_PATH` | Token store path | `/var/lib/arcaflow-mcp/tokens` | No |
| `ARCAFLOW_MCP_AUDIT_STORE_PATH` | Audit log path | `/var/lib/arcaflow-mcp/audit` | No |
| `ARCAFLOW_MCP_USAGE_STORE_PATH` | Usage stats path | `/var/lib/arcaflow-mcp/usage` | No |
| `ARCAFLOW_MCP_WORKSPACE_ROOT` | Tenant workspaces root | `/var/lib/arcaflow-mcp/workspaces` | No |
| `ARCAFLOW_MCP_LISTEN_ADDR` | HTTP listen address | `:8080` | No |

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

### Development Compose

```yaml
# deploy/container/compose.yml
version: '3.8'

services:
  analysis:
    image: quay.io/arcalot/arcaflow-mcp-analysis:latest
    container_name: arcaflow-analysis
    ports:
      - "8081:8081"
    volumes:
      - analysis-data:/var/lib/arcaflow-analysis
    environment:
      - ANALYSIS_PORT=8081
      - ANALYSIS_HOST=0.0.0.0
      - LOG_LEVEL=INFO
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s

  mcp-server:
    image: quay.io/arcalot/arcaflow-mcp-server:latest
    container_name: arcaflow-mcp-server
    ports:
      - "8080:8080"
    depends_on:
      analysis:
        condition: service_healthy
    volumes:
      - mcp-data:/var/lib/arcaflow-mcp
      - ./workflows:/workflows:ro
    environment:
      - ARCAFLOW_MCP_ADMIN_TOKEN=${ARCAFLOW_MCP_ADMIN_TOKEN:-changeme}
      - ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://analysis:8081
    command: ["--server"]
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 10s

volumes:
  mcp-data:
  analysis-data:
```

### Usage

```bash
# Set admin token
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"

# Start services
docker compose -f deploy/container/compose.yml up -d

# View logs
docker compose -f deploy/container/compose.yml logs -f

# Stop services
docker compose -f deploy/container/compose.yml down
```

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
