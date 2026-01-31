# Tenant-Master BFF (Backend for Frontend)

A high-performance Go REST API service that bridges the Tenant-Master UI and Kubernetes API.

## Features

- **Dual-mode Operation**: Mock mode (local testing) and k8s mode (cluster integration)
- **Full CRUD**: List, create, read, update, delete Tenant CRDs
- **JWT Authentication**: Token-based auth (optional, configured via `JWT_SECRET`)
- **CORS Support**: For local development and cross-domain requests
- **Kubernetes Integration**: Uses controller-runtime client for type-safe API interaction
- **Real-time Metrics**: Proxies Prometheus metrics and tenant metrics
- **Kubeconfig Export**: Gold-tier vCluster kubeconfig retrieval
- **RBAC**: ServiceAccount with minimal required permissions

## Build

```bash
cd bff
go mod tidy
go build -o bff ./...
```

## Run

### Mock Mode (Local Development)

```bash
export BFF_MODE=mock
export BFF_PORT=8080
./bff
```

Mock mode reads from `examples/tenants/` directory and does not require Kubernetes.

### Kubernetes Mode (Production)

```bash
export BFF_MODE=k8s
export BFF_PORT=8080
export JWT_SECRET=<secure-random-value>
./bff
```

Runs inside a Kubernetes cluster using in-cluster config and controller-runtime client.

### Environment Variables

```bash
BFF_MODE=k8s                    # "mock" or "k8s"
BFF_PORT=8080                   # Listen port
JWT_SECRET=<random-value>       # JWT secret for auth (optional)
```

## API Endpoints

### Authentication

All endpoints (except `/health`) require JWT bearer token in `Authorization` header if `JWT_SECRET` is set:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/tenants
```

### Endpoints

#### List Tenants

```bash
GET /api/v1/tenants
```

**Response:**
```json
[
  {
    "name": "acme-payments",
    "tier": "Silver",
    "owner": "payments@acme.corp",
    "state": "Ready",
    "namespace": "tenant-acme-payments",
    "cpu": "4000m",
    "memory": "8Gi",
    "createdAt": "2024-01-31T10:00:00Z"
  }
]
```

#### Get Tenant Details

```bash
GET /api/v1/tenants/:name
```

#### Create Tenant

```bash
POST /api/v1/tenants
Content-Type: application/json

{
  "name": "new-tenant",
  "tier": "Silver",
  "owner": "owner@company.com",
  "resources": {
    "cpu": "4000m",
    "memory": "8Gi"
  },
  "network": {
    "allowInternetAccess": false,
    "whitelistedServices": ["kube-system/coredns"]
  }
}
```

#### Update Tenant

```bash
PATCH /api/v1/tenants/:name
Content-Type: application/json

{
  "tier": "Gold"
}
```

#### Delete Tenant

```bash
DELETE /api/v1/tenants/:name
```

#### Get Metrics

```bash
GET /api/v1/tenants/:name/metrics
```

**Response:**
```json
{
  "tenant": "acme-payments",
  "metrics": {
    "cpu_usage": "250m",
    "memory_usage": "512Mi",
    "last_provisioning_seconds": 42.5,
    "active": true
  }
}
```

#### Export Kubeconfig (Gold Tier)

```bash
GET /api/v1/tenants/:name/kubeconfig
```

**Response:** Raw kubeconfig YAML

#### Health Check

```bash
GET /health
```

## Deployment

### Using Helm (with Tenant-Master operator Helm chart)

The BFF is deployed alongside the operator. Update your Helm values:

```yaml
bff:
  enabled: true
  mode: k8s
  replicas: 2
  image: amartyaa/tenant-master-bff:latest
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

### Using kubectl

```bash
# 1. Create namespace
kubectl create namespace tenant-master-system

# 2. Generate JWT secret (replace with secure random value)
kubectl create secret generic bff-jwt-secret \
  -n tenant-master-system \
  --from-literal=secret=$(openssl rand -base64 32)

# 3. Apply RBAC and Deployment
kubectl apply -f bff/rbac.yaml
```

### Required Permissions (RBAC)

The BFF ServiceAccount requires:
- `platform.io/v1alpha1/tenants` (get, list, create, update, patch, delete, watch)
- `platform.io/v1alpha1/tenants/status` (get, update, patch)
- `v1/secrets` (get, list) - for kubeconfig export
- `v1/namespaces` (get, list) - for tenant info

## Docker Build

```dockerfile
# Build stage
FROM golang:1.25 AS builder
WORKDIR /src
COPY . .
RUN go mod tidy && go build -o /tmp/bff .

# Runtime stage
FROM gcr.io/distroless/base:nonroot
COPY --from=builder /tmp/bff /bff
USER nonroot
EXPOSE 8080
CMD ["/bff"]
```

```bash
docker build -t amartyaa/tenant-master-bff:latest .
docker push amartyaa/tenant-master-bff:latest
```

## Development

### Testing with curl

```bash
# List tenants (mock mode)
curl http://localhost:8080/api/v1/tenants | jq

# Create tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-tenant",
    "tier": "Silver",
    "owner": "test@example.com",
    "resources": {"cpu": "2000m", "memory": "4Gi"}
  }' | jq

# Get tenant details
curl http://localhost:8080/api/v1/tenants/test-tenant | jq
```

### Architecture

- **main.go**: Server setup, middleware (CORS, auth), route registration
- **handlers.go**: Request handlers with mock/k8s mode dispatch
  - Mock mode: reads YAML from file system
  - k8s mode: uses controller-runtime client for API calls
- **Middleware**:
  - CORS: Allow cross-domain requests
  - JWT Auth: Validates bearer tokens
  - Health: Unauthenticated health check

### Controller-Runtime Integration

Uses `sigs.k8s.io/controller-runtime` for Kubernetes API interaction:

```go
import "sigs.k8s.io/controller-runtime/pkg/client"

// In-cluster client initialization
cfg, _ := rest.InClusterConfig()
k8sClient, _ := client.New(cfg, client.Options{})

// List tenants
list := &unstructured.UnstructuredList{}
list.SetGroupVersionKind(...)
k8sClient.List(ctx, list)
```

## Future Enhancements

- [ ] Webhook validation for tenant creation
- [ ] Audit logging for all mutations
- [ ] Metrics export for Prometheus
- [ ] WebSocket support for real-time updates
- [ ] OIDC integration for enterprise auth
- [ ] Role-based access control (RBAC) per tenant
- [ ] Tenant quota enforcement
- [ ] Backup/restore APIs
