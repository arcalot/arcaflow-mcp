# Kubernetes Deployment

Deploy Arcaflow MCP Server on Kubernetes for scalable, production-grade deployments.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Persistence Requirements](#persistence-requirements)
5. [Deployment Manifests](#deployment-manifests)
6. [Scaling and High Availability](#scaling-and-high-availability)
7. [Secrets Management](#secrets-management)
8. [Service Configuration](#service-configuration)
9. [Ingress and TLS](#ingress-and-tls)
10. [Monitoring](#monitoring)
11. [Troubleshooting](#troubleshooting)

---

## Overview

Kubernetes deployment provides:

- **Scalability**: Horizontal pod autoscaling
- **High Availability**: Multi-replica deployments
- **Rolling Updates**: Zero-downtime deployments
- **Resource Management**: CPU/memory requests and limits
- **Self-Healing**: Automatic pod restarts
- **Load Balancing**: Built-in service load balancing

**Deployment Patterns:**

- **Single-Instance**: Simple deployment for development/testing
- **Multi-Replica**: High availability with shared storage
- **StatefulSet**: For stateful workloads (future)

---

## Prerequisites

**Kubernetes Cluster:**

- Kubernetes 1.24+ cluster
- kubectl configured
- Persistent Volume provisioner (for storage)

**Storage:**

- ReadWriteOnce (RWO) PersistentVolumes for single-instance
- ReadWriteMany (RWX) PersistentVolumes for multi-replica (or shared storage backend)

**Optional:**

- Ingress controller (nginx, Traefik, etc.) for external access
- cert-manager for automated TLS certificate management
- Metrics Server for autoscaling

---

## Quick Start

### Single-Instance Deployment

```bash
# Create namespace
kubectl create namespace arcaflow-mcp

# Apply manifests
kubectl apply -f deploy/kubernetes/

# Check status
kubectl get pods -n arcaflow-mcp

# Expose service (for testing)
kubectl port-forward -n arcaflow-mcp svc/arcaflow-mcp-server 8080:8080
```

### Access the Server

```bash
# Get admin token
export ARCAFLOW_MCP_ADMIN_TOKEN=$(kubectl get secret -n arcaflow-mcp arcaflow-admin-token -o jsonpath='{.data.token}' | base64 -d)

# Test MCP protocol
curl -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  http://localhost:8080/health
```

---

## Persistence Requirements

### Storage Classes

**Development/Testing:**

- **hostPath**: Local node storage (not recommended for production)
- **local**: Local persistent volumes

**Production:**

- **NFS**: Network File System (ReadWriteMany support)
- **Ceph/GlusterFS**: Distributed storage
- **Cloud Providers**: EBS (AWS), Persistent Disk (GCP), Azure Disk

### Required PersistentVolumeClaims

1. **MCP Server Data**: Tenant records, tokens, audit logs, usage stats
2. **Tenant Workspaces**: Isolated workspaces per tenant
3. **Analysis Engine Data**: SQLite database or PostgreSQL connection

---

## Deployment Manifests

### Namespace

```yaml
# deploy/kubernetes/00-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: arcaflow-mcp
  labels:
    app.kubernetes.io/name: arcaflow-mcp
```

### ConfigMap

```yaml
# deploy/kubernetes/01-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: arcaflow-mcp-config
  namespace: arcaflow-mcp
data:
  config.yaml: |
    server:
      enabled: true
      listen_addr: ":8080"
    
    analysis:
      http_url: "http://arcaflow-analysis-service:8081"
    
    tenancy:
      workspace_root: "/var/lib/arcaflow-mcp/workspaces"
      tenant_store_path: "/var/lib/arcaflow-mcp/tenants"
    
    auth:
      token_store_path: "/var/lib/arcaflow-mcp/tokens"
    
    audit:
      store_path: "/var/lib/arcaflow-mcp/audit"
    
    usage:
      store_path: "/var/lib/arcaflow-mcp/usage"
```

### Secret (Admin Token)

```bash
# Generate admin token
ADMIN_TOKEN=$(openssl rand -base64 32)

# Create secret
kubectl create secret generic arcaflow-admin-token \
  --namespace=arcaflow-mcp \
  --from-literal=token="$ADMIN_TOKEN"
```

Or via YAML:

```yaml
# deploy/kubernetes/02-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: arcaflow-admin-token
  namespace: arcaflow-mcp
type: Opaque
data:
  token: <base64-encoded-token>
```

### PersistentVolumeClaims

```yaml
# deploy/kubernetes/03-pvc.yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: arcaflow-mcp-data
  namespace: arcaflow-mcp
spec:
  accessModes:
    - ReadWriteOnce  # Use ReadWriteMany for multi-replica
  resources:
    requests:
      storage: 10Gi
  storageClassName: standard  # Adjust to your cluster's storage class

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: arcaflow-analysis-data
  namespace: arcaflow-mcp
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: standard
```

### Python Analysis Engine Deployment

```yaml
# deploy/kubernetes/04-analysis-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: arcaflow-analysis
  namespace: arcaflow-mcp
  labels:
    app: arcaflow-analysis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: arcaflow-analysis
  template:
    metadata:
      labels:
        app: arcaflow-analysis
    spec:
      containers:
      - name: analysis
        image: quay.io/arcalot/arcaflow-mcp-analysis:v1.0.0
        ports:
        - containerPort: 8081
          name: http
        env:
        - name: ANALYSIS_PORT
          value: "8081"
        - name: ANALYSIS_HOST
          value: "0.0.0.0"
        - name: LOG_LEVEL
          value: "INFO"
        volumeMounts:
        - name: data
          mountPath: /var/lib/arcaflow-analysis
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: arcaflow-analysis-data

---
apiVersion: v1
kind: Service
metadata:
  name: arcaflow-analysis-service
  namespace: arcaflow-mcp
spec:
  selector:
    app: arcaflow-analysis
  ports:
  - name: http
    port: 8081
    targetPort: 8081
  type: ClusterIP
```

### MCP Server Deployment

```yaml
# deploy/kubernetes/05-mcp-server-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
  labels:
    app: arcaflow-mcp-server
spec:
  replicas: 1  # Set to 2+ for HA (requires ReadWriteMany PVC)
  selector:
    matchLabels:
      app: arcaflow-mcp-server
  template:
    metadata:
      labels:
        app: arcaflow-mcp-server
    spec:
      containers:
      - name: server
        image: quay.io/arcalot/arcaflow-mcp-server:v1.0.0
        args: ["--server"]
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: ARCAFLOW_MCP_ADMIN_TOKEN
          valueFrom:
            secretKeyRef:
              name: arcaflow-admin-token
              key: token
        - name: ARCAFLOW_MCP_ANALYSIS_HTTP_URL
          value: "http://arcaflow-analysis-service:8081"
        - name: ARCAFLOW_MCP_TENANT_STORE_PATH
          value: "/var/lib/arcaflow-mcp/tenants"
        - name: ARCAFLOW_MCP_TOKEN_STORE_PATH
          value: "/var/lib/arcaflow-mcp/tokens"
        - name: ARCAFLOW_MCP_AUDIT_STORE_PATH
          value: "/var/lib/arcaflow-mcp/audit"
        - name: ARCAFLOW_MCP_USAGE_STORE_PATH
          value: "/var/lib/arcaflow-mcp/usage"
        - name: ARCAFLOW_MCP_WORKSPACE_ROOT
          value: "/var/lib/arcaflow-mcp/workspaces"
        volumeMounts:
        - name: data
          mountPath: /var/lib/arcaflow-mcp
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: arcaflow-mcp-data
      - name: config
        configMap:
          name: arcaflow-mcp-config

---
apiVersion: v1
kind: Service
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  selector:
    app: arcaflow-mcp-server
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  type: ClusterIP
  # For multi-replica with SSE, enable session affinity:
  # sessionAffinity: ClientIP
  # sessionAffinityConfig:
  #   clientIP:
  #     timeoutSeconds: 3600
```

---

## Scaling and High Availability

### Clustered Deployment Requirements

Running multiple replicas requires shared storage for tenant data.

**Shared Storage Options:**

1. **ReadWriteMany (RWX) PersistentVolume**:
   - NFS, CephFS, GlusterFS
   - All pods access the same storage

2. **External Storage Backend**:
   - PostgreSQL for tenant/token/audit data
   - Shared NFS for tenant workspaces

### Multi-Replica Deployment

```yaml
# deploy/kubernetes/05-mcp-server-deployment-ha.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  replicas: 3  # High availability
  selector:
    matchLabels:
      app: arcaflow-mcp-server
  template:
    metadata:
      labels:
        app: arcaflow-mcp-server
    spec:
      affinity:
        # Spread pods across nodes
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - arcaflow-mcp-server
              topologyKey: kubernetes.io/hostname
      containers:
      - name: server
        # ... (same as single-instance)
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: arcaflow-mcp-data-rwx  # Must be ReadWriteMany

---
apiVersion: v1
kind: Service
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  selector:
    app: arcaflow-mcp-server
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  type: ClusterIP
  # CRITICAL: Enable session affinity for SSE support
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600
```

### Horizontal Pod Autoscaler

```yaml
# deploy/kubernetes/06-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: arcaflow-mcp-server-hpa
  namespace: arcaflow-mcp
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: arcaflow-mcp-server
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## Secrets Management

### Using Kubernetes Secrets

```bash
# Create secret from file
kubectl create secret generic arcaflow-admin-token \
  --namespace=arcaflow-mcp \
  --from-file=token=./admin-token.txt

# Create secret from literal
kubectl create secret generic arcaflow-admin-token \
  --namespace=arcaflow-mcp \
  --from-literal=token="$(openssl rand -base64 32)"
```

### Using External Secrets Operator

```yaml
# deploy/kubernetes/02-external-secret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: arcaflow-admin-token
  namespace: arcaflow-mcp
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: arcaflow-admin-token
  data:
  - secretKey: token
    remoteRef:
      key: arcaflow/admin-token
      property: value
```

### Using Sealed Secrets

```bash
# Install kubeseal
# https://sealed-secrets.netlify.app/

# Create sealed secret
echo -n "your-admin-token" | kubectl create secret generic arcaflow-admin-token \
  --namespace=arcaflow-mcp \
  --dry-run=client \
  --from-file=token=/dev/stdin \
  -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

# Apply sealed secret
kubectl apply -f sealed-secret.yaml
```

---

## Service Configuration

### ClusterIP (Internal Only)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  type: ClusterIP
  selector:
    app: arcaflow-mcp-server
  ports:
  - port: 8080
    targetPort: 8080
```

### NodePort (External Access)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  type: NodePort
  selector:
    app: arcaflow-mcp-server
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30080  # Access via <node-ip>:30080
```

### LoadBalancer (Cloud Provider)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  type: LoadBalancer
  selector:
    app: arcaflow-mcp-server
  ports:
  - port: 8080
    targetPort: 8080
```

---

## Ingress and TLS

### Ingress with nginx

```yaml
# deploy/kubernetes/07-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: arcaflow-mcp-ingress
  namespace: arcaflow-mcp
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    # For SSE support:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.ingress.kubernetes.io/affinity: "cookie"
    nginx.ingress.kubernetes.io/session-cookie-name: "arcaflow-session"
    nginx.ingress.kubernetes.io/session-cookie-max-age: "3600"
spec:
  ingressClassName: nginx
  rules:
  - host: arcaflow-mcp.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: arcaflow-mcp-server
            port:
              number: 8080
  tls:
  - hosts:
    - arcaflow-mcp.example.com
    secretName: arcaflow-mcp-tls
```

### TLS with cert-manager

```yaml
# deploy/kubernetes/08-certificate.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: arcaflow-mcp-cert
  namespace: arcaflow-mcp
spec:
  secretName: arcaflow-mcp-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - arcaflow-mcp.example.com
```

---

## Monitoring

### ServiceMonitor (Prometheus)

```yaml
# deploy/kubernetes/09-servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
  labels:
    app: arcaflow-mcp-server
spec:
  selector:
    matchLabels:
      app: arcaflow-mcp-server
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

### PodMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: arcaflow-mcp-server
  namespace: arcaflow-mcp
spec:
  selector:
    matchLabels:
      app: arcaflow-mcp-server
  podMetricsEndpoints:
  - port: http
    path: /metrics
```

---

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl get pods -n arcaflow-mcp

# Check pod events
kubectl describe pod -n arcaflow-mcp <pod-name>

# Check logs
kubectl logs -n arcaflow-mcp <pod-name>

# Check previous logs (if crashed)
kubectl logs -n arcaflow-mcp <pod-name> --previous
```

### PersistentVolumeClaim Pending

```bash
# Check PVC status
kubectl get pvc -n arcaflow-mcp

# Check PVC events
kubectl describe pvc -n arcaflow-mcp arcaflow-mcp-data

# Check available storage classes
kubectl get storageclass

# Manually create PV (if dynamic provisioning unavailable)
kubectl apply -f pv.yaml
```

### Service Not Accessible

```bash
# Check service
kubectl get svc -n arcaflow-mcp

# Check endpoints
kubectl get endpoints -n arcaflow-mcp arcaflow-mcp-server

# Port forward for debugging
kubectl port-forward -n arcaflow-mcp svc/arcaflow-mcp-server 8080:8080
```

### Multi-Replica Issues

**SSE Session Binding:**

```bash
# Verify session affinity enabled
kubectl get svc -n arcaflow-mcp arcaflow-mcp-server -o yaml | grep sessionAffinity

# Check if all pods are receiving traffic
kubectl logs -n arcaflow-mcp -l app=arcaflow-mcp-server --tail=10
```

---

## Related Documentation

- [Container Deployment](container.md) - Docker/Podman deployment
- [Authentication](authentication.md) - Configure authentication
- [TLS Configuration](tls.md) - TLS best practices

---

*For Helm chart deployment (future), see [Helm Charts](helm.md).*
