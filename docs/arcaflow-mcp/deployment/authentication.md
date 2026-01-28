# Authentication Setup

**Configuring Secure Authentication for Server Mode**

Server mode requires bearer token authentication for all operations. This guide covers token management, tenant provisioning, and security best practices.

---

## Overview

Server mode uses a two-tier authentication model:

1. **Admin Token** - For tenant and token management (administrative operations)
2. **Tenant Tokens** - For MCP operations (workflow loading, result analysis)

All requests must include `Authorization: Bearer <token>` header.

---

## Prerequisites

Before configuring authentication:

- Arcaflow MCP server installed
- Server mode configured (see [Server Mode Setup](../usage/server-mode.md))
- Access to server configuration files or environment variables
- Secure method for generating and storing tokens

---

## Step 1: Configure Admin Token

### Generate Secure Admin Token

**For production, use a cryptographically secure random token:**

```bash
# Option 1: OpenSSL (recommended)
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -hex 32)"

# Option 2: UUID
export ARCAFLOW_MCP_ADMIN_TOKEN="$(uuidgen)"

# Option 3: Password manager
# Use your organization's password manager to generate
```

**For development/testing:**

```bash
# Timestamp-based (not for production!)
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-admin-token-$(date +%s)"
```

### Configure Server

**Via environment variable (recommended):**

```bash
export ARCAFLOW_MCP_ADMIN_TOKEN="your-secure-admin-token"
./arcaflow-mcp --mode server --address :8080
```

**Via configuration file:**

```yaml
# config.yaml
auth:
  admin_token: "your-secure-admin-token"
  token_store_path: "/var/lib/arcaflow-mcp/tokens.json"
```

```bash
./arcaflow-mcp --config config.yaml
```

### Store Admin Token Securely

**Development:**
- Environment variable in shell profile
- Local secrets file (with restricted permissions)
- Development secrets manager

**Production:**
- Kubernetes Secrets
- HashiCorp Vault
- Cloud provider secrets manager (AWS Secrets Manager, Azure Key Vault, etc.)
- Encrypted configuration management

**Never:**
- Commit tokens to git repositories
- Share tokens via email or chat
- Log tokens in application logs
- Expose tokens in URLs or query parameters

---

## Step 2: Create Tenants

Tenants represent teams, projects, or user groups with isolated workspaces.

### Create a Tenant

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<'EOF'
{
  "tenant_id": "team-platform",
  "display_name": "Platform Engineering Team",
  "metadata": {
    "department": "engineering",
    "cost_center": "eng-001",
    "contact": "platform-team@example.com"
  }
}
EOF
```

**Response:**
```json
{
  "tenant_id": "team-platform",
  "display_name": "Platform Engineering Team",
  "metadata": {...},
  "created_at": "2026-01-28T12:00:00Z"
}
```

### Tenant ID Guidelines

**Requirements:**
- Must match `[A-Za-z0-9_.-]` pattern
- 1-128 characters in length
- Unique across all tenants
- Immutable after creation

**Best Practices:**
- Use descriptive names: `team-qa` not `t1`
- Include organizational context: `dept-eng-team-platform`
- Consistent naming convention across organization
- Document tenant purpose in display name or metadata

---

## Step 3: Create Tenant Tokens

Tenant tokens are scoped to a single tenant and used for MCP operations.

### Create Token

```bash
curl -X POST http://localhost:8080/admin/tenants/team-platform/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<'EOF'
{}
EOF
```

**Response (save this token securely):**
```json
{
  "token": "tnt_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "tenant_id": "team-platform",
  "created_at": "2026-01-28T12:05:00Z"
}
```

### Token Distribution

**For individual users:**
1. Create one token per user
2. Send via secure channel (password manager, encrypted email)
3. User configures their AI client with the token
4. Document which token belongs to which user

**For service accounts:**
1. Create dedicated tokens for automation/CI
2. Store in secrets management system
3. Use descriptive tenant IDs: `service-ci-pipeline`
4. Rotate regularly per security policy

### Token Storage

**Client Side:**
```json
// Claude Desktop config: ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "arcaflow": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "http://server:8080/mcp",
        "-H", "Authorization: Bearer tnt_..."
      ]
    }
  }
}
```

**Automation:**
```bash
# Environment variable
export ARCAFLOW_MCP_TOKEN="tnt_a1b2c3d4..."

# Use in scripts
curl -H "Authorization: Bearer $ARCAFLOW_MCP_TOKEN" ...
```

---

## Step 4: Verify Authentication

### Test Admin Access

```bash
# List all tenants (requires admin token)
curl http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

**Expected:** List of all tenants (or empty array if none created yet)

### Test Tenant Access

```bash
# Test tenant token with ping
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer tnt_a1b2c3d4..." \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ping"
}
EOF
```

**Expected:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {}
}
```

### Test Invalid Token

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid-token" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"ping"}
EOF
```

**Expected:**
```json
{
  "error": "unauthorized"
}
```

---

## Token Management

### List Tenant Tokens

```bash
curl http://localhost:8080/admin/tenants/team-platform/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

**Response:**
```json
{
  "tokens": [
    {
      "token": "tnt_a1b2c3d4...",
      "created_at": "2026-01-28T12:05:00Z"
    },
    {
      "token": "tnt_x9y8z7...",
      "created_at": "2026-01-28T14:30:00Z"
    }
  ]
}
```

### Revoke Token

When a user leaves the team or token is compromised:

```bash
curl -X DELETE \
  http://localhost:8080/admin/tenants/team-platform/tokens/tnt_a1b2c3d4... \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

**Response:**
```json
{
  "revoked": true,
  "tenant_id": "team-platform",
  "token": "tnt_a1b2c3d4..."
}
```

### Token Rotation

**Best Practice:** Rotate tokens regularly (e.g., every 90 days).

**Procedure:**
1. Create new token for tenant
2. Distribute new token to users
3. Users update their AI client configurations
4. Verify new token works
5. Revoke old token

**Automation example:**
```bash
# Create new token
NEW_TOKEN=$(curl -X POST http://localhost:8080/admin/tenants/team-platform/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{}' | jq -r '.token')

# Update secrets manager
vault kv put secret/arcaflow/tenant-platform token="$NEW_TOKEN"

# Wait for users to migrate (e.g., 7 days)
sleep $((7 * 24 * 3600))

# Revoke old token
curl -X DELETE \
  http://localhost:8080/admin/tenants/team-platform/tokens/$OLD_TOKEN \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

---

## Security Best Practices

### Token Generation

**✅ Do:**
- Use cryptographically secure random generation
- Generate tokens with at least 256 bits of entropy
- Use different tokens for different tenants
- Store tokens in secure secrets management

**❌ Don't:**
- Use predictable patterns (sequential IDs, timestamps alone)
- Reuse tokens across environments (dev/staging/prod)
- Share tokens between users
- Store tokens in plain text files

### Token Transmission

**✅ Do:**
- Use HTTPS/TLS for all production deployments
- Send tokens via secure channels (secrets manager, encrypted email)
- Use environment variables or secrets managers
- Clear tokens from shell history when testing

**❌ Don't:**
- Send tokens via plain HTTP
- Include tokens in URLs or query parameters
- Log tokens to application logs
- Store tokens in version control

### Token Storage

**Development:**
```bash
# Environment variable (session only)
export ARCAFLOW_MCP_TOKEN="tnt_..."

# Or from file (chmod 600)
export ARCAFLOW_MCP_TOKEN="$(cat ~/.arcaflow-token)"
chmod 600 ~/.arcaflow-token
```

**Production:**
```bash
# Kubernetes Secret
kubectl create secret generic arcaflow-token \
  --from-literal=token="tnt_..."

# Mount in pod
env:
  - name: ARCAFLOW_MCP_TOKEN
    valueFrom:
      secretKeyRef:
        name: arcaflow-token
        key: token
```

### Access Control

**Principle of Least Privilege:**
- Create separate tenants for different teams/projects
- Use service accounts for automation (not personal tokens)
- Revoke tokens immediately when user leaves
- Regular audit of active tokens

**Monitoring:**
- Review audit logs for unauthorized access attempts
- Monitor usage patterns for anomalies
- Alert on failed authentication attempts
- Track token usage per tenant

---

## Multi-User Setup Example

### Scenario

Platform Engineering team with 5 engineers needs shared access.

### Setup Procedure

**1. Create tenant:**
```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "tenant_id": "team-platform",
  "display_name": "Platform Engineering",
  "metadata": {"size": "5", "lead": "alice@example.com"}
}
EOF
```

**2. Create tokens (one per engineer):**
```bash
# Repeat for each engineer
for user in alice bob carol dave eve; do
  curl -X POST http://localhost:8080/admin/tenants/team-platform/tokens \
    -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
    -d '{}' | jq -r '.token' > token-$user.txt
  chmod 600 token-$user.txt
done
```

**3. Distribute tokens securely:**
```bash
# Option 1: Store in secrets manager
for user in alice bob carol dave eve; do
  vault kv put secret/arcaflow/users/$user \
    token="$(cat token-$user.txt)"
done

# Option 2: Send via encrypted email
# (Implementation depends on your email system)

# Clean up token files
rm token-*.txt
```

**4. Users configure their AI clients:**

Each engineer adds to Claude Desktop config:
```json
{
  "mcpServers": {
    "arcaflow": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "https://arcaflow-mcp.example.com/mcp",
        "-H", "Authorization: Bearer tnt_<their-token>",
        "-H", "Content-Type: application/json"
      ]
    }
  }
}
```

**5. Verify access:**

Each user tests:
```bash
curl -X POST https://arcaflow-mcp.example.com/mcp \
  -H "Authorization: Bearer tnt_<their-token>" \
  -d '{"jsonrpc":"2.0","id":1,"method":"ping"}'
```

---

## Troubleshooting

### "Unauthorized" Errors

**Symptom:** `401 Unauthorized` or `403 Forbidden` responses

**Possible Causes:**
1. Token not provided or malformed
2. Token revoked or expired
3. Wrong tenant token for operation
4. Admin token used where tenant token required (or vice versa)

**Solutions:**
```bash
# Verify token in request
curl -v http://localhost:8080/mcp \
  -H "Authorization: Bearer $TOKEN" \
  ... | grep Authorization

# List valid tokens for tenant
curl http://localhost:8080/admin/tenants/team-platform/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"

# Check audit logs for authentication failures
curl "http://localhost:8080/admin/audit?action=auth_failure&limit=20" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN"
```

### Admin Token Not Set

**Symptom:** Server fails to start with "admin token must be set"

**Solution:**
```bash
# Verify environment variable is set
echo $ARCAFLOW_MCP_ADMIN_TOKEN

# If empty, set it
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -hex 32)"

# Or use configuration file
cat > config.yaml <<EOF
auth:
  admin_token: "$(openssl rand -hex 32)"
EOF

./arcaflow-mcp --config config.yaml --mode server
```

### Token Store Permission Errors

**Symptom:** "permission denied" when accessing token store

**Solution:**
```bash
# Check token store path
echo $ARCAFLOW_MCP_TOKEN_STORE_PATH

# Verify directory permissions
ls -la $(dirname "$ARCAFLOW_MCP_TOKEN_STORE_PATH")

# Fix permissions
chmod 700 $(dirname "$ARCAFLOW_MCP_TOKEN_STORE_PATH")
chmod 600 "$ARCAFLOW_MCP_TOKEN_STORE_PATH"
```

---

## Security Recommendations

### Token Security

**Generation:**
- Minimum 32 bytes (256 bits) of randomness
- Use cryptographically secure random number generators
- Never use predictable patterns

**Storage:**
- Encrypt at rest
- Restrict file permissions (600 or stricter)
- Use secrets management in production
- Audit access to token storage

**Transmission:**
- Always use HTTPS in production (see [TLS Configuration](tls.md))
- Never send tokens via unencrypted channels
- Avoid including in URLs (use headers)
- Clear from logs and shell history

**Rotation:**
- Establish rotation policy (e.g., every 90 days)
- Automate rotation where possible
- Provide grace period for migration
- Revoke old tokens after grace period

### Tenant Isolation

**Workspace:**
- Verify tenant workspace isolation is enforced
- Check filesystem permissions on tenant directories
- Audit cross-tenant access attempts
- Use quotas to prevent disk exhaustion

**Operations:**
- Tenant tokens can only access their own workspace
- No cross-tenant API calls permitted
- Audit logs track tenant boundaries
- Regular security review of isolation

### Monitoring and Alerting

**Monitor:**
- Failed authentication attempts (potential attacks)
- Unusual usage patterns (compromised tokens)
- Token creation/revocation events
- Admin API access

**Alert on:**
- Repeated authentication failures from single IP
- Token used from unexpected locations
- Spike in admin API usage
- Workspace quota violations

---

## Integration with Identity Providers

### Future: OAuth/OIDC Support

Planned for future releases:
- OAuth 2.0 / OpenID Connect integration
- SSO with enterprise identity providers
- Automatic tenant provisioning
- Group-based access control

**Current Workaround:**
- Use external proxy (e.g., oauth2-proxy) for OAuth
- Proxy validates OAuth tokens
- Proxy adds tenant header for MCP server
- MCP server trusts proxy-provided tenant identity

---

## Related Documentation

- **[Server Mode Setup](../usage/server-mode.md)** - Complete server deployment guide
- **[Multi-Tenancy](../concepts/multi-tenancy.md)** - Multi-user architecture
- **[TLS Configuration](tls.md)** - Encrypting network traffic
- **[Configuration Reference](../usage/configuration.md)** - All auth configuration options

---

[← Back to Documentation Index](../index.md)
