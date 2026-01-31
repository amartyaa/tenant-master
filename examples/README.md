# Tenant-Master Operator: Example Tenants

This directory contains example Tenant CRD manifests demonstrating how to provision and configure tenants in a multi-tenant Kubernetes cluster using the Tenant-Master operator.

## Overview

The Tenant-Master operator provides three isolation tiers, each suited to different use cases:

| Tier | Isolation | Use Case | Namespace | NetworkPolicy | vCluster | Cost |
|------|-----------|----------|-----------|---------------|----------|------|
| **Bronze** | Soft | Dev/Test | Shared | ❌ | ❌ | $$$ |
| **Silver** | Namespace | Production Internal | Dedicated | ✅ | ❌ | $$$$ |
| **Gold** | Complete | Enterprise/SaaS | Dedicated | ✅ | ✅ | $$$$$$ |

## Quick Start

### 1. Create a Bronze Tier Tenant (Development)

```bash
kubectl apply -f examples/tenants/bronze-tier-example.yaml
```

Bronze tier tenants share a namespace and have soft RBAC isolation. Best for:
- Development environments
- CI/CD testing
- Non-production workloads
- Cost-sensitive projects

**Resource Limits:**
- CPU: 1-2 cores
- Memory: 2-4 GB

**Network:**
- Full internet access allowed
- No NetworkPolicy enforcement

### 2. Create a Silver Tier Tenant (Production Internal)

```bash
kubectl apply -f examples/tenants/silver-tier-example.yaml
```

Silver tier tenants get their own namespace with hard isolation. Best for:
- Internal production services
- Microservices backends
- Business-critical applications
- Stable, long-running workloads

**Resource Limits:**
- CPU: 4-8 cores
- Memory: 8-16 GB

**Network:**
- No internet access by default
- Explicit whitelisting of allowed services
- NetworkPolicy enforcement

**Example manifests:**
- `acme-payments-silver`: Payment processing service
- `acme-analytics-silver`: Data analytics pipeline
- `nightly-batch-jobs-silver`: Scheduled batch processing (suspended by default)
- `platform-monitoring-silver`: Observability stack

### 3. Create a Gold Tier Tenant (Enterprise/SaaS)

```bash
kubectl apply -f examples/tenants/gold-tier-example.yaml
```

Gold tier tenants get a dedicated virtual Kubernetes cluster (vCluster). Best for:
- SaaS customer deployments
- Highly regulated industries (HIPAA, PCI-DSS)
- External customer workloads
- Complete control plane isolation required

**Resource Limits:**
- CPU: 6-16 cores
- Memory: 12-32 GB

**Network:**
- Strict egress controls
- vCluster kernel isolation
- Complete API endpoint isolation

**Example manifests:**
- `customer-alpha-gold`: Standard SaaS customer
- `healthtech-hipaa-gold`: Healthcare compliance (HIPAA)
- `fintech-pci-gold`: Financial services (PCI-DSS)
- `alpha-staging-vcluster-gold`: Staging/pre-production vCluster

## Manifest Structure

Each tenant manifest includes:

```yaml
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: tenant-name              # Unique identifier
  namespace: default             # Tenant CRD location (not tenant's namespace)
spec:
  tier: Silver                   # Bronze | Silver | Gold
  owner: owner@company.com       # Contact for notifications
  resources:
    cpu: 4000m                   # CPU in millicores
    memory: 8Gi                  # Memory in bytes
    storageClass: premium-ssd    # Storage class for PVCs
  network:
    allowInternetAccess: false   # Default: true (Bronze), false (Silver/Gold)
    whitelistedServices:         # Egress destinations for Silver/Gold
      - namespace/service
  allowTierMigration: false      # Prevent downgrades
  suspend: false                 # Scale to zero for cost savings
```

## Common Operations

### Create a New Tenant

```bash
kubectl apply -f my-tenant.yaml
```

### Check Tenant Status

```bash
kubectl get tenants
kubectl describe tenant <tenant-name>
```

### Watch Provisioning Progress

```bash
kubectl get tenants -w
# or
kubectl logs -f -l app=tenant-operator -n tenant-master-system
```

### Suspend a Tenant (Scale to Zero)

```bash
kubectl patch tenant <tenant-name> -p '{"spec":{"suspend":true}}'
```

### Resume a Suspended Tenant

```bash
kubectl patch tenant <tenant-name> -p '{"spec":{"suspend":false}}'
```

### Access Gold Tier Kubeconfig

For Gold tier tenants, retrieve the auto-generated kubeconfig:

```bash
kubectl get secret <tenant-name>-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d > kubeconfig.yaml
export KUBECONFIG=kubeconfig.yaml
kubectl cluster-info  # Verify vCluster connectivity
```

### Upgrade Silver → Gold

To migrate from Silver to Gold tier (namespace isolation → vCluster):

```bash
kubectl patch tenant <tenant-name> -p '{"spec":{"tier":"Gold"}}'
```

The operator will:
1. Provision a new vCluster
2. Create admin kubeconfig Secret
3. Keep the original namespace for data migration
4. Provide API endpoint for tenant access

## Tier Comparison Examples

### Example 1: Development Workflow

```bash
# Start development with Bronze tier
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: my-app-dev
spec:
  tier: Bronze
  owner: dev@company.com
  resources:
    cpu: 1000m
    memory: 2Gi
EOF

# Later, promote to Silver for stable production
kubectl patch tenant my-app-dev -p '{"spec":{"tier":"Silver"}}'
```

### Example 2: Regulatory Compliance

```bash
# Healthcare application requiring HIPAA isolation
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: patient-data-service
spec:
  tier: Gold                           # Complete isolation required
  owner: compliance-officer@health.com
  resources:
    cpu: 12000m
    memory: 24Gi
    storageClass: encrypted-premium
  network:
    allowInternetAccess: false
    whitelistedServices:
      - kube-system/coredns
      - monitoring/audit-logger      # Compliance logging required
EOF
```

### Example 3: Cost Optimization

```bash
# Batch processing job: suspend when not needed
kubectl apply -f - <<EOF
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: monthly-reports
spec:
  tier: Silver
  owner: finance@company.com
  resources:
    cpu: 8000m
    memory: 16Gi
  suspend: true                       # Don't run continuously
EOF

# Enable before scheduled execution
kubectl patch tenant monthly-reports -p '{"spec":{"suspend":false}}'

# Disable after job completes
kubectl patch tenant monthly-reports -p '{"spec":{"suspend":true}}'
```

## Troubleshooting

### Tenant stuck in "Provisioning" state

```bash
# Check operator logs
kubectl logs -l app=tenant-operator -n tenant-master-system -f

# Inspect tenant status
kubectl describe tenant <tenant-name>

# Check resource availability
kubectl describe nodes
```

### Cannot access tenant workloads

For **Silver tier**: Verify NetworkPolicy whitelisting
```bash
kubectl get networkpolicies -n <tenant-namespace>
```

For **Gold tier**: Retrieve and use the vCluster kubeconfig
```bash
kubectl get secret <tenant-name>-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d > kubeconfig.yaml
export KUBECONFIG=kubeconfig.yaml
```

### Resource quota exceeded

```bash
# Check current usage
kubectl describe resourcequota -n <tenant-namespace>

# Upgrade tenant resources (if on Silver tier upgrade to Gold)
kubectl patch tenant <tenant-name> -p '{"spec":{"resources":{"cpu":"8000m","memory":"16Gi"}}}'
```

## Best Practices

1. **Choose the Right Tier**
   - Bronze: Temporary, non-critical workloads
   - Silver: Stable internal services
   - Gold: External customers, regulated data

2. **Resource Planning**
   - Bronze: Generous limits, low cost
   - Silver: Balanced requests/limits
   - Gold: Premium resources with SLA guarantees

3. **Network Security**
   - Always whitelist only required services
   - Use DNS names over IP addresses
   - Review egress rules quarterly

4. **Compliance**
   - Gold tier for PCI-DSS, HIPAA, SOC2 workloads
   - Enable audit logging for regulatory tenants
   - Document network policies for compliance audits

5. **Cost Optimization**
   - Suspend non-critical Silver tenants during off-hours
   - Use Bronze for temporary environments
   - Right-size resources per actual usage
   - Monitor metrics for over-provisioning

## Next Steps

- Deploy a tenant: `kubectl apply -f examples/tenants/silver-tier-example.yaml`
- Monitor operator: `kubectl logs -f -l app=tenant-operator -n tenant-master-system`
- Check provisioning: `kubectl get tenants -w`
- Access metrics: `kubectl port-forward -n tenant-master-system svc/tenant-operator-metrics 8080:8080`

For more information, see the [Tenant-Master README](../../README.md).
