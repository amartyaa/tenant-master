# Tenant-Master Quick Start Guide

## 5-Minute Setup

This guide walks you through deploying Tenant-Master and creating your first tenant in a Kubernetes cluster.

### Prerequisites

- Kubernetes 1.28+ cluster (Kind, EKS, GKE, or similar)
- `kubectl` configured to access your cluster
- Docker (for building container image, optional if using pre-built)

### Step 1: Deploy the Operator

```bash
# Apply CRD
kubectl apply -f config/crd/tenant_crd.yaml

# Apply RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding)
kubectl apply -f config/rbac/rbac.yaml

# Apply webhook configuration
kubectl apply -f config/webhook/webhook.yaml

# Deploy the operator
kubectl apply -f config/manager/manager.yaml

# Verify installation
kubectl get ns tenant-system
kubectl get deployment -n tenant-system
kubectl wait --for=condition=available --timeout=300s deployment/tenant-master -n tenant-system
```

Expected output:
```
NAME              READY   UP-TO-DATE   AVAILABLE   AGE
tenant-master     1/1     1            1           5s
```

### Step 2: Create Your First Tenant

#### Silver Tier (Namespace-Isolated)

```bash
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: acme-corp
spec:
  tier: Silver
  owner: admin@acme.com
  resources:
    cpu: "2000m"
    memory: "4Gi"
  network:
    allowInternetAccess: false
    whitelistedServices:
    - "shared-services/auth"
EOF
```

Check tenant status:
```bash
kubectl get tenants
kubectl describe tenant acme-corp

# Expected output:
# Status:
#   State: Ready
#   Namespace: tenant-acme-corp
```

Verify tenant namespace:
```bash
kubectl get ns tenant-acme-corp
kubectl get serviceaccount -n tenant-acme-corp
kubectl get resourcequota -n tenant-acme-corp
kubectl get networkpolicy -n tenant-acme-corp
```

#### Gold Tier (Virtual Cluster)

```bash
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: bigbank-enterprise
spec:
  tier: Gold
  owner: platform-admin@bigbank.com
  resources:
    cpu: "8000m"
    memory: "16Gi"
  network:
    allowInternetAccess: false
    whitelistedServices:
    - "shared-services/auth"
    - "monitoring/prometheus"
EOF
```

Monitor provisioning (wait for Gold tier to complete, ~45s):
```bash
kubectl get tenant bigbank-enterprise -w
# Watch Status transition: Provisioning → Ready
```

### Step 3: Deploy Workload to Tenant

```bash
# Deploy a simple app to the tenant namespace
kubectl apply -f - -n tenant-acme-corp <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-sample
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
EOF
```

Verify workload:
```bash
kubectl get pods -n tenant-acme-corp
kubectl logs -n tenant-acme-corp -l app=nginx
```

### Step 4: Test Network Isolation

#### Test 1: Verify Intra-Tenant Communication (Should Work)

```bash
# Deploy a test pod in the tenant namespace
kubectl run -it --rm -n tenant-acme-corp test --image=alpine:latest -- sh

# Inside the pod, try to reach the nginx service in the same namespace
/ # wget -O- http://nginx-sample.default.svc.cluster.local
  # OR (using tenant namespace)
/ # wget -O- http://nginx-sample.tenant-acme-corp.svc.cluster.local

# Result: Should succeed (200 OK)
```

#### Test 2: Verify Cross-Tenant Isolation (Should Fail)

```bash
# Create two tenants
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: tenant-a
spec:
  tier: Silver
  owner: admin-a@example.com
  network: {}
---
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: tenant-b
spec:
  tier: Silver
  owner: admin-b@example.com
  network: {}
EOF

# Deploy nginx to both tenants
for ns in tenant-tenant-a tenant-tenant-b; do
  kubectl apply -f - -n $ns <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
EOF
done

# Wait for both deployments to be ready
kubectl wait --for=condition=ready pod -l app=nginx -n tenant-tenant-a --timeout=60s
kubectl wait --for=condition=ready pod -l app=nginx -n tenant-tenant-b --timeout=60s

# Try to reach tenant-b nginx from tenant-a (should timeout/fail)
kubectl run -it --rm -n tenant-tenant-a test --image=alpine:latest --timeout=10s -- \
  wget -O- http://nginx.tenant-tenant-b.svc.cluster.local

# Result: Connection timeout (NetworkPolicy blocks cross-namespace traffic)
```

### Step 5: Monitor Operator Metrics

```bash
# Port-forward to metrics service
kubectl port-forward -n tenant-system svc/tenant-master-metrics 8080:8080

# In another terminal, check metrics
curl http://localhost:8080/metrics | grep tenant_

# Example output:
# tenant_provisioning_seconds_bucket{tier="Silver",le="1"} 1
# tenant_provisioning_seconds_bucket{tier="Silver",le="2"} 1
# active_tenants_count{tier="Silver"} 2
# active_tenants_count{tier="Gold"} 1
```

### Step 6: Delete a Tenant

```bash
# Delete a tenant
kubectl delete tenant acme-corp

# Watch cleanup (namespace and child resources removed)
kubectl get ns -w | grep tenant-acme-corp

# Verify cleanup complete
kubectl get ns tenant-acme-corp
# Result: "NotFound" (namespace deleted)
```

---

## Common Tasks

### List All Tenants

```bash
kubectl get tenants
kubectl get tenants -o wide
kubectl get tenants -o custom-columns=NAME:.metadata.name,TIER:.spec.tier,STATE:.status.state,NAMESPACE:.status.namespace,OWNER:.spec.owner
```

### Describe a Tenant

```bash
kubectl describe tenant acme-corp
```

### Check Tenant Logs

```bash
# Operator logs
kubectl logs -n tenant-system -f deployment/tenant-master

# Filter by tenant name
kubectl logs -n tenant-system deployment/tenant-master | grep "acme-corp"
```

### Access Tenant Kubeconfig (Gold Tier)

```bash
# Extract kubeconfig from secret
TENANT_NAME="bigbank-enterprise"
NAMESPACE="tenant-${TENANT_NAME}"
SECRET_NAME="${TENANT_NAME}-kubeconfig"

kubectl get secret -n $NAMESPACE $SECRET_NAME -o jsonpath='{.data.kubeconfig}' | base64 -d > /tmp/vcluster-kubeconfig.yaml

# Use the kubeconfig to access the virtual cluster
export KUBECONFIG=/tmp/vcluster-kubeconfig.yaml
kubectl cluster-info
kubectl get nodes  # Should show vCluster nodes
```

### Upgrade Resource Quota

```bash
# Edit tenant resource spec
kubectl patch tenant acme-corp --type merge -p '{"spec":{"resources":{"cpu":"4000m","memory":"8Gi"}}}'

# Verify ResourceQuota updated
kubectl get resourcequota -n tenant-acme-corp -o yaml
```

### Add Whitelisted Service

```bash
# Add "logging/elasticsearch" to whitelisted services
kubectl patch tenant acme-corp --type merge -p '{"spec":{"network":{"whitelistedServices":["shared-services/auth","logging/elasticsearch"]}}}'

# Verify NetworkPolicy updated
kubectl get networkpolicy -n tenant-acme-corp default-deny-all -o yaml
```

### Scale Tenant to Zero (Sleep Mode)

```bash
# Set suspend=true (future feature)
kubectl patch tenant acme-corp --type merge -p '{"spec":{"suspend":true}}'

# Result: All deployments in tenant namespace scaled to 0 replicas
```

---

## Troubleshooting

### Tenant Stuck in "Provisioning"

```bash
# Check operator logs
kubectl logs -n tenant-system deployment/tenant-master -f

# Check tenant events
kubectl describe tenant acme-corp | tail -20

# Check for specific errors
kubectl get tenant acme-corp -o jsonpath='{.status.lastError}'
```

### Webhook Validation Failures

```bash
# Test with invalid tier (should be rejected)
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: invalid-test
spec:
  tier: InvalidTier  # ← Invalid
  owner: admin@example.com
EOF

# Result: Error from server: error when creating "STDIN": admission webhook "vtenant.platform.io" denied the request: ...
```

### NetworkPolicy Not Applied

```bash
# Verify NetworkPolicy exists
kubectl get networkpolicy -n tenant-acme-corp

# Inspect NetworkPolicy details
kubectl get networkpolicy -n tenant-acme-corp default-deny-all -o yaml

# Verify a pod can't reach external service
kubectl run -it --rm -n tenant-acme-corp test --image=alpine:latest -- \
  timeout 5 wget -O- http://example.com
# Result: Connection timeout (expected, no internet egress)
```

### RBAC Issues

```bash
# Verify ServiceAccount exists
kubectl get sa -n tenant-acme-corp

# Check RoleBinding
kubectl get rolebinding -n tenant-acme-corp

# Test RBAC permissions
kubectl auth can-i get pods --as=system:serviceaccount:tenant-acme-corp:acme-corp-sa -n tenant-acme-corp
# Result: yes

kubectl auth can-i get pods --as=system:serviceaccount:tenant-acme-corp:acme-corp-sa -n tenant-bigbank-enterprise
# Result: no (expected, RBAC isolated to namespace)
```

---

## Next Steps

- **Review Architecture:** See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design
- **Explore Examples:** Check [config/samples/](config/samples/) for more examples
- **Configure Monitoring:** Integrate with Prometheus and Grafana
- **Implement Phase 2:** Integrate Helm SDK for true vCluster deployment
- **Add Custom Webhooks:** Extend validation/mutation logic for your use case
- **Production Deployment:** Use managed registries, webhook certificates, and HA setup

---

## Support

- **Documentation:** See [README.md](README.md)
- **Issues:** Report bugs or request features via GitHub
- **Examples:** Check [config/samples/](config/samples/) for example configurations

---

**Happy multi-tenant clustering! 🚀**
