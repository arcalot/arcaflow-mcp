# Container Deployment Files

This directory contains deployment configurations for running Arcaflow MCP in containers.

> **🚨 Pre-Release Note (Before v0.1.0)**: Container tags change with each commit to main.
> 
> **Get current tag:**
> ```bash
> export TAG=$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash)
> echo "Current tag: $TAG"
> ```
> 
> After v0.1.0, use `:latest` or version tags like `:v1.0.0`

## Quick Start

### Using Docker Compose

```bash
# Step 1: Get current development tag (before v0.1.0)
export TAG=$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash)
echo "Using tag: $TAG"

# Step 2: Generate admin token
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
echo "Admin token: $ARCAFLOW_MCP_ADMIN_TOKEN"
echo "IMPORTANT: Save this token for API access!"

# Step 3: Optional - Create workflow directory if you plan to mount workflows
mkdir -p ./workflows

# Step 4: Start both components
docker compose up -d

# Check status
docker compose ps
docker compose logs -f

# Verify both components are healthy
curl http://localhost:8081/health  # Analysis engine
curl http://localhost:8080/health  # MCP server
```

### Using Podman Compose

```bash
# Generate admin token
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
echo "Admin token: $ARCAFLOW_MCP_ADMIN_TOKEN"

# Optional: Create workflow directory if you plan to mount workflows
mkdir -p ./workflows

# Start both components
podman-compose up -d

# Check status
podman-compose ps
podman-compose logs -f

# Verify both components are healthy
curl http://localhost:8081/health  # Analysis engine
curl http://localhost:8080/health  # MCP server
```

## Configuration

### Environment Variables

You can set environment variables directly or use a `.env` file:

```bash
# Option 1: Create .env file from example
cp .env.example .env
# Edit .env with your values
nano .env

# Option 2: Export directly
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
export TAG="latest"
```

**Available Variables:**

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `ARCAFLOW_MCP_ADMIN_TOKEN` | Admin authentication token | **Yes** | - |
| `TAG` | Container image tag | No | `latest` |
| `IMAGE_REPO` | Container registry | No | `quay.io/arcalot` |
| `LOG_LEVEL` | Logging level | No | `INFO` |
| `WORKFLOW_DIR` | Local workflow directory | No | `./workflows` |

### Using Specific Tags

By default, the compose file uses `latest`. To use a specific version:

```bash
# Use latest (default - no TAG needed)
docker compose up -d

# Use specific release version
export TAG="v1.0.0"
docker compose up -d

# Use specific commit build
export TAG="main-abc1234"
docker compose up -d
```

### Finding Available Tags

Query Quay.io to see all available tags:

```bash
# Using skopeo
skopeo list-tags docker://quay.io/arcalot/arcaflow-mcp-server

# Using Quay.io API
curl -s 'https://quay.io/api/v1/repository/arcalot/arcaflow-mcp-server/tag/?limit=100' | \
  jq -r '.tags[].name' | sort -V

# Browse tags: https://quay.io/repository/arcalot/arcaflow-mcp-server?tab=tags
```

## Verifying Deployment

### Quick Health Checks

```bash
# Check Analysis Engine (Component 1)
curl http://localhost:8081/health
# Expected: {"status":"healthy"}

# Check MCP Server (Component 2)
curl http://localhost:8080/health
# Expected: {"status":"healthy","version":"..."}

# View logs if needed
docker compose logs -f analysis    # Analysis engine logs
docker compose logs -f mcp-server  # MCP server logs
```

### Complete MCP Protocol Test

Verify the MCP server is working with a complete handshake:

```bash
# Step 1: Initialize
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# Expected: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25",...}}

# Step 2: Complete initialization
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","method":"initialized"}'

# Step 3: List tools (verifies server is ready)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

# Expected: {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}
```

### Troubleshooting Health Checks

**If health checks fail:**

```bash
# Check container status
docker compose ps

# Check detailed logs
docker compose logs analysis --tail 50
docker compose logs mcp-server --tail 50

# Check if ports are bound
sudo lsof -i :8080
sudo lsof -i :8081

# Restart if needed
docker compose restart
```

## Stopping Services

```bash
# Stop containers (preserves data)
docker compose stop

# Remove containers (preserves volumes)
docker compose down

# Remove everything including volumes (CAUTION: deletes data)
docker compose down -v
```

## Data Persistence

The compose file creates named volumes for persistent data:

- `mcp-data`: MCP server data (tenants, tokens, audit logs, usage stats)
- `analysis-data`: Analysis engine database

To backup data:

```bash
# List volumes
docker volume ls | grep arcaflow

# Backup MCP data
docker run --rm -v arcaflow-mcp-data:/data -v $PWD:/backup alpine tar czf /backup/mcp-data-backup.tar.gz -C /data .

# Backup analysis data
docker run --rm -v arcaflow-mcp-analysis-data:/data -v $PWD:/backup alpine tar czf /backup/analysis-data-backup.tar.gz -C /data .
```

## Customization

### Custom Workflow Directory

Mount a local directory with workflows:

```bash
export WORKFLOW_DIR="/path/to/workflows"
docker compose up -d
```

### Custom Database (Production)

For production, use PostgreSQL instead of SQLite:

1. Add PostgreSQL service to compose.yml
2. Set `DATABASE_URL` environment variable for analysis service:
   ```yaml
   environment:
     - DATABASE_URL=postgresql://user:pass@postgres:5432/arcaflow
   ```

### Custom Configuration File

To use a custom config file for the MCP server:

```bash
# Create config.yaml
cat > config.yaml <<EOF
server:
  enabled: true
  listen_addr: ":8080"
analysis:
  http_url: "http://analysis:8081"
EOF

# Mount in compose.yml (add to mcp-server volumes)
volumes:
  - ./config.yaml:/app/config.yaml:ro
  
# Update command
command: ["--mode", "server", "--config", "/app/config.yaml"]
```

## Troubleshooting

### Containers won't start

**Check logs:**
```bash
docker compose logs
docker compose logs analysis  # Analysis engine only
docker compose logs mcp-server  # MCP server only
```

**Common issues:**

1. **Missing ARCAFLOW_MCP_ADMIN_TOKEN**
   ```
   Error: ARCAFLOW_MCP_ADMIN_TOKEN is required
   ```
   **Fix:** Set the environment variable before running compose
   ```bash
   export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
   ```

2. **Port already in use**
   ```
   Error: bind: address already in use
   ```
   **Fix:** Check what's using ports 8080 or 8081
   ```bash
   # Check port usage
   sudo lsof -i :8080
   sudo lsof -i :8081
   # Or change ports in compose.yml
   ports:
     - "8090:8080"  # Map to different host port
   ```

3. **Insufficient permissions on volumes**
   ```
   Error: permission denied
   ```
   **Fix:** Check SELinux labels or volume permissions
   ```bash
   # For Podman with SELinux
   mkdir -p ./workflows
   chcon -Rt svirt_sandbox_file_t ./workflows
   # Or run with :z flag
   - ./workflows:/workflows:ro,z
   ```

4. **Workflow directory doesn't exist**
   ```
   Error: no such file or directory: ./workflows
   ```
   **Fix:** Create the directory or comment out the volume mount
   ```bash
   mkdir -p ./workflows
   ```

### Analysis engine not reachable

**Verify network connectivity:**
```bash
# From MCP server container
docker compose exec mcp-server wget -qO- http://analysis:8081/health

# From host machine
curl http://localhost:8081/health
```

**Common issues:**
- Analysis engine failed health check (check logs)
- Network isolation issue (verify both containers are on `arcaflow` network)
- Firewall blocking container communication

### Data not persisting

**Check volumes:**
```bash
docker volume ls | grep arcaflow
docker volume inspect arcaflow-mcp_mcp-data
docker volume inspect arcaflow-mcp_analysis-data
```

**Verify volume mounts:**
```bash
docker compose exec mcp-server ls -la /var/lib/arcaflow-mcp
docker compose exec analysis ls -la /var/lib/arcaflow-analysis
```

### Health checks failing

**Analysis engine:**
```bash
# Check if service is listening
docker compose exec analysis netstat -tlnp | grep 8081

# Test health endpoint
docker compose exec analysis curl -f http://localhost:8081/health
```

**MCP server:**
```bash
# Check if service is listening
docker compose exec mcp-server netstat -tlnp | grep 8080

# Test health endpoint
docker compose exec mcp-server wget --spider -q http://localhost:8080/health
```

## Related Documentation

- [Container Deployment Guide](../../docs/arcaflow-mcp/deployment/container.md)
- [Getting Started](../../docs/arcaflow-mcp/getting-started.md)
- [Server Mode Usage](../../docs/arcaflow-mcp/usage/server-mode.md)
