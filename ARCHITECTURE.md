# Tenant-Master Architecture Document

## Overview

Tenant-Master is a production-grade Kubernetes Operator that implements **Hard Multi-Tenancy** through infrastructure-level isolation. This document describes the architecture, design decisions, and implementation details.

## Design Principles

### 1. Infrastructure-Level Isolation (Not App-Level)

**Problem:** Most SaaS platforms implement multi-tenancy at the application layer using `tenant_id` columns, row-level security, or shared databases. This is inherently fragile:
- Data leakage risks
- Performance isolation failures ("Noisy Neighbor")
- Compliance gaps (SOC2/HIPAA require physical separation)

**Solution:** Tenant-Master enforces isolation at the **infrastructure layer** using Kubernetes primitives:
- Dedicated namespaces for hard resource boundaries
- NetworkPolicies for zero-trust networking
- RBAC for access control
- ResourceQuotas to prevent resource hogging

### 2. Declarative, GitOps-Friendly Design

Tenants are managed via CRDs (Custom Resource Definitions), enabling:
- Version control of tenant configurations
- Audit trails of all changes
- GitOps workflows
- Infrastructure-as-Code practices

### 3. Tier-Based Flexibility

Three isolation tiers allow cost-effective scaling:
- **Bronze:** Soft isolation for free trials (lowest cost)
- **Silver:** Hard isolation via namespace (standard offering)
- **Gold:** Extreme isolation via virtual cluster (premium offering)

This enables monetization aligned with customer requirements.

### 4. Reconciliation-Based (Not Imperativ)

The operator uses a **reconciliation loop** model:
- Watches for Tenant CRD changes
- Compares desired state (spec) vs actual state (cluster)
- Takes corrective actions to converge
- Provides natural recovery from failures

This differs from imperative scripting, which is brittle and doesn't recover from drift.

## Component Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ tenant-system namespace (Operator Infrastructure)      │ │
│  │                                                         │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │ Tenant-Master Deployment                        │   │ │
│  │  │                                                  │   │ │
│  │  │  Pod: tenant-master-xxxxx                       │   │ │
│  │  │  ├─ Container: manager                          │   │ │
│  │  │  │  ├─ TenantReconciler                        │   │ │
│  │  │  │  ├─ Webhook Server (9443)                   │   │ │
│  │  │  │  └─ Metrics Exporter (8080)                 │   │ │
│  │  │  └─ Volumes:                                   │   │ │
│  │  │     └─ webhook-certs (webhook TLS)             │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                         │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │ Webhook Service (ClusterIP)                     │   │ │
│  │  │ ├─ webhook-service:443 → manager:9443         │   │ │
│  │  │ └─ (MutatingWebhookConfiguration + Validating) │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                         │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │ Metrics Service (ClusterIP)                     │   │ │
│  │  │ └─ tenant-master-metrics:8080 → manager:8080  │   │ │
│  │  │    (scraped by Prometheus)                      │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                         │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ Tenant Namespaces (Dynamic)                            │ │
│  │                                                         │ │
│  │  ┌──────────────────────────────────────────────────┐  │ │
│  │  │ tenant-acme-corp (Silver Tier)                  │  │ │
│  │  │                                                  │  │ │
│  │  │  ├─ Namespace                                   │  │ │
│  │  │  │  └─ Labels: tier=Silver, owner=admin@...   │  │ │
│  │  │  │                                              │  │ │
│  │  │  ├─ ServiceAccount: acme-corp-sa              │  │ │
│  │  │  ├─ Role: acme-corp-admin                     │  │ │
│  │  │  ├─ RoleBinding: acme-corp-admin-binding      │  │ │
│  │  │  ├─ ResourceQuota: acme-corp-quota            │  │ │
│  │  │  │  └─ CPU: 4000m, Memory: 8Gi               │  │ │
│  │  │  └─ NetworkPolicy: default-deny-all          │  │ │
│  │  │     ├─ Ingress: Allow from same NS only      │  │ │
│  │  │     └─ Egress: Allow DNS + whitelisted svc  │  │ │
│  │  │                                                │  │ │
│  │  │  Tenant User Workloads:                        │  │ │
│  │  │  ├─ Deployment: acme-api-server              │  │ │
│  │  │  ├─ StatefulSet: acme-db                     │  │ │
│  │  │  ├─ ConfigMaps, Secrets, PVCs                │  │ │
│  │  │  └─ ... (any K8s workload)                   │  │ │
│  │  └──────────────────────────────────────────────────┘  │ │
│  │                                                         │ │
│  │  ┌──────────────────────────────────────────────────┐  │ │
│  │  │ tenant-bigbank-enterprise (Gold Tier)           │  │ │
│  │  │                                                  │  │ │
│  │  │  ├─ (Silver tier resources as above)           │  │ │
│  │  │  │                                              │  │ │
│  │  │  ├─ vCluster StatefulSet                       │  │ │
│  │  │  │  ├─ etcd pod (Kubernetes data store)       │  │ │
│  │  │  │  ├─ apiserver pod (K8s API server)         │  │ │
│  │  │  │  ├─ controller-manager pod                 │  │ │
│  │  │  │  └─ scheduler pod                          │  │ │
│  │  │  │                                              │  │ │
│  │  │  └─ Secret: bigbank-enterprise-kubeconfig     │  │ │
│  │  │     └─ data.kubeconfig (vCluster kubeconfig)  │  │ │
│  │  │                                                  │  │ │
│  │  │  Tenant's Virtual Cluster (accessible via kubeconfig):
│  │  │  ├─ tenant-api-server                          │  │ │
│  │  │  ├─ tenant-db                                  │  │ │
│  │  │  ├─ Custom CRDs (installed by tenant)         │  │ │
│  │  │  └─ ... (full Kubernetes cluster)             │  │ │
│  │  └──────────────────────────────────────────────────┘  │ │
│  │                                                         │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

## Reconciliation Flow

### High-Level Reconciliation Loop

```
User Creates Tenant CRD
    ↓
Kubernetes API Server (Etcd)
    ↓
TenantReconciler (watches for Tenant events)
    ↓
┌─ Fetch Tenant object
│
├─ Is DeletionTimestamp set?
│  ├─ YES → Handle Deletion (run finalizers, cleanup)
│  └─ NO → Continue
│
├─ Ensure Finalizer is set (for cleanup on delete)
│
├─ Update Status to "Provisioning"
│
├─ Reconcile based on Tier:
│  ├─ Bronze: Document as minimal isolation (future implementation)
│  ├─ Silver: reconcileSilverTier()
│  │  ├─ ensureNamespace()
│  │  ├─ ensureResourceQuota()
│  │  ├─ ensureRBAC()
│  │  └─ ensureNetworkPolicy()
│  └─ Gold: reconcileGoldTier()
│     ├─ Call reconcileSilverTier() first
│     ├─ ensureVCluster()
│     └─ ensureKubeconfigSecret()
│
├─ If reconciliation succeeded:
│  ├─ Update Status to "Ready"
│  └─ Record metrics (provisioning time, active tenants)
│
└─ If reconciliation failed:
   ├─ Update Status to "Failed"
   ├─ Record error in status.lastError
   └─ Requeue after 30 seconds
```

### Detailed: ensureNamespace()

```
ensureNamespace(tenant) {
  namespaceName = "tenant-" + tenant.name
  
  Build Namespace object:
  - metadata.name = namespaceName
  - metadata.labels:
    - tenant.platform.io/name = tenant.name
    - tenant.platform.io/tier = Silver|Gold|Bronze
    - tenant.platform.io/owner = admin@example.com
    - app.kubernetes.io/managed-by = tenant-master
  
  Set OwnerReference to Tenant (for garbage collection)
  
  Call controllerutil.CreateOrUpdate():
    - If not exists: Create
    - If exists: Update labels (idempotent)
  
  Store namespaceName in tenant.status.namespace
}
```

### Detailed: ensureResourceQuota()

```
ensureResourceQuota(tenant) {
  Parse spec.resources.cpu and spec.resources.memory
  
  Create ResourceQuota object:
  - metadata.namespace = tenant namespace
  - metadata.name = tenant.name + "-quota"
  
  spec.hard:
    - requests.cpu = parsed CPU
    - requests.memory = parsed Memory
    - limits.cpu = parsed CPU
    - limits.memory = parsed Memory
    - pods = 100 (prevent DoS via pod spam)
  
  Set OwnerReference to Tenant
  Call controllerutil.CreateOrUpdate()
}
```

### Detailed: ensureNetworkPolicy()

```
ensureNetworkPolicy(tenant) {
  Create NetworkPolicy with:
  
  PolicyTypes: [Ingress, Egress]
  PodSelector: {} (apply to all pods in namespace)
  
  Ingress rules:
    - Allow from pods in same namespace (intra-tenant traffic)
  
  Egress rules:
    - Allow DNS (kube-system/coredns:53 UDP)
    - For each whitelisted service:
      - Allow to namespace/service
    - If allowInternetAccess=true:
      - Allow to 0.0.0.0/0
  
  Result: Default-deny model with explicit allow list
}
```

## Data Flow: Tenant Creation

### Step 1: User Applies Tenant CRD

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
    cpu: "4000m"
    memory: "8Gi"
  network:
    whitelistedServices:
    - "shared-services/auth-api"
EOF
```

### Step 2: Admission Webhooks Intercept

**Mutating Webhook** (if tier was missing, default to Silver, etc.):
```
Request: Tenant CRD
  ↓
Mutating Webhook: mtenant.platform.io
  ├─ Default tier to Silver (not needed in this example)
  ├─ Normalize owner to lowercase
  ├─ Set default resources (not needed in this example)
  ↓
Enhanced Tenant CRD (possibly modified)
  ↓
Kubernetes API Server (stores in etcd)
```

**Validating Webhook** (validates spec constraints):
```
Request: Tenant CRD (after mutation)
  ↓
Validating Webhook: vtenant.platform.io
  ├─ Verify tier is one of: Bronze, Silver, Gold ✓
  ├─ Verify owner is valid email ✓
  ├─ Verify CPU/Memory quantities are parseable ✓
  ├─ Check for unsafe tier downgrades (N/A on create)
  ↓
ACCEPTED or REJECTED
  ↓
Kubernetes API Server (stores or rejects)
```

### Step 3: Reconciliation Loop Detects Change

```
Tenant-Master TenantReconciler watches for Tenant events
  ↓
Event: Tenant "acme-corp" ADDED
  ↓
Reconcile(req.NamespacedName = "acme-corp")
  ├─ Fetch Tenant from etcd ✓
  ├─ Check DeletionTimestamp (not set) ✓
  ├─ Ensure Finalizer added ✓
  ├─ Tier = Silver → call reconcileSilverTier()
  │  ├─ ensureNamespace()
  │  │  └─ Create "tenant-acme-corp" namespace with labels
  │  ├─ ensureResourceQuota()
  │  │  └─ Create ResourceQuota: 4CPU, 8GB memory limits
  │  ├─ ensureRBAC()
  │  │  ├─ Create ServiceAccount: "acme-corp-sa"
  │  │  ├─ Create Role: "acme-corp-admin" (full namespace admin)
  │  │  └─ Create RoleBinding: links SA to Role
  │  └─ ensureNetworkPolicy()
  │     └─ Create default-deny NetworkPolicy with whitelisted egress
  │
  ├─ Update Status:
  │  ├─ state = "Ready"
  │  ├─ namespace = "tenant-acme-corp"
  │  └─ lastUpdateTime = now
  │
  └─ Return (no error → success)
      ↓
      Record Metrics:
        ├─ tenant_provisioning_seconds{tier="Silver"} = 0.5s
        └─ active_tenants_count{tier="Silver"} ++
```

### Step 4: User Verifies Tenant

```bash
# List tenants
$ kubectl get tenants
NAME         TIER    STATE   NAMESPACE              OWNER
acme-corp    Silver  Ready   tenant-acme-corp       admin@acme.com

# Describe tenant
$ kubectl describe tenant acme-corp
Status:
  State: Ready
  Namespace: tenant-acme-corp
  Last Update Time: 2025-01-30T10:23:45Z

# Verify namespace and resources
$ kubectl get ns tenant-acme-corp
NAME                 STATUS   AGE
tenant-acme-corp     Active   2s

$ kubectl get resourcequota -n tenant-acme-corp
NAME             AGE   REQUEST.CPU   REQUEST.MEMORY
acme-corp-quota  2s    4000m         8Gi

$ kubectl get networkpolicy -n tenant-acme-corp
NAME               POD-SELECTOR   AGE
default-deny-all   <none>         2s
```

## Webhook Design

### Mutating Webhook (mtenant.platform.io)

**Purpose:** Apply sensible defaults and normalize inputs

**Triggers:** CREATE, UPDATE on Tenant CRDs

**Mutations:**
1. If `spec.tier` is empty → default to `Silver`
2. Normalize `spec.owner` to lowercase (for consistency)
3. If `spec.resources.cpu` is empty → default to `"1000m"`
4. If `spec.resources.memory` is empty → default to `"1Gi"`

**Example:**
```yaml
# Input
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: test
spec:
  owner: ADMIN@EXAMPLE.COM
  # tier and resources omitted

# After Mutation
apiVersion: platform.io/v1alpha1
kind: Tenant
metadata:
  name: test
spec:
  tier: Silver                    # Defaulted
  owner: admin@example.com        # Lowercased
  resources:
    cpu: "1000m"                  # Defaulted
    memory: "1Gi"                 # Defaulted
```

### Validating Webhook (vtenant.platform.io)

**Purpose:** Enforce constraints and prevent unsafe operations

**Triggers:** CREATE, UPDATE on Tenant CRDs

**Validations:**
1. `spec.tier` must be one of: Bronze, Silver, Gold
2. `spec.owner` must be a valid email address
3. `spec.resources.cpu` and `spec.resources.memory` must be valid K8s quantities
4. **Tier Downgrade Prevention:** If updating and new tier < old tier (less isolated), reject unless `spec.allowTierMigration=true`

**Example: Unsafe Downgrade**
```yaml
# Current: Gold tier
tier: Gold

# Update attempt: Downgrade to Bronze
tier: Bronze
allowTierMigration: false  # ← Unsafe!

# Result: REJECTED by validating webhook
# Error: "unsafe tier downgrade: Gold -> Bronze. Set spec.allowTierMigration=true to proceed (DATA MAY BE LOST)"
```

**Example: Allowed Downgrade**
```yaml
# Current: Gold tier
tier: Gold

# Update with explicit flag:
tier: Bronze
allowTierMigration: true  # ← Explicit acknowledgment

# Result: ACCEPTED
```

## Finalizers & Cleanup

### Why Finalizers?

Without finalizers, deleting a Tenant CRD would immediately trigger garbage collection (via OwnerReferences), removing the namespace and all child resources. This is risky because:
- No time for data backup/snapshot
- No audit trail of cleanup
- No opportunity for graceful shutdown

### Finalizer Workflow

```
User: kubectl delete tenant acme-corp
  ↓
Kubernetes API Server:
  ├─ Set metadata.deletionTimestamp
  ├─ Retain Tenant in etcd (don't delete yet)
  └─ Trigger watchers (TenantReconciler)
  ↓
TenantReconciler.Reconcile():
  ├─ Detect DeletionTimestamp is set
  ├─ Call handleDeletion()
  │  ├─ Update status to "Terminating"
  │  ├─ Perform cleanup:
  │  │  ├─ (Future) Snapshot tenant data
  │  │  ├─ (Future) Archive tenant metadata
  │  │  └─ (For now) Log cleanup action
  │  ├─ Remove finalizer from Tenant
  │  └─ Update Tenant in API Server
  ↓
Kubernetes API Server:
  ├─ Finalizer list is now empty
  ├─ Trigger garbage collection (OwnerReferences)
  ├─ Delete Namespace (tenant-acme-corp)
  │  └─ Cascading delete: all child resources deleted
  └─ Delete Tenant CRD from etcd
  ↓
Result: Clean removal of all resources
```

## Error Handling & Recovery

### Reconciliation Failure Scenarios

**Scenario 1: Namespace Creation Fails**
```
reconcileSilverTier():
  ensureNamespace() → Error: "quota exceeded"
    ↓
  reconcile() returns error
    ↓
  Update status:
    - state = "Failed"
    - lastError = "namespace creation failed: quota exceeded"
    ↓
  Requeue after 30 seconds (exponential backoff)
    ↓
  Next reconciliation attempt (after cluster admin adds capacity)
    ↓
  Namespace created successfully
    ↓
  state = "Ready"
```

**Scenario 2: Manual NetworkPolicy Modification (Drift)**
```
Operator creates NetworkPolicy (default-deny)
  ↓
User manually modifies: kubectl edit networkpolicy default-deny-all
  ├─ Removes egress rule for "shared-services/auth-api"
  ↓
watchNetworkPolicies() detects change:
  ├─ Compares current spec vs desired spec
  ├─ Detects drift
  ├─ Applies correction (recreate with desired spec)
  ↓
NetworkPolicy reverted to correct state
```

**Scenario 3: Controller Restart**
```
Operator Pod crashes
  ↓
Kubernetes restarts pod
  ↓
Controller starts with empty in-memory state
  ↓
Controller re-lists all Tenant CRDs from API Server
  ↓
For each Tenant: Reconcile to desired state
  ├─ If namespace missing: Create it
  ├─ If ResourceQuota missing: Create it
  ├─ If NetworkPolicy changed: Update it
  ↓
All Tenants converge to desired state
```

## Security Architecture

### Defense in Depth

```
Layer 1: RBAC (Role-Based Access Control)
  ├─ Each tenant's ServiceAccount can only access their namespace
  └─ Prevent cross-tenant privilege escalation

Layer 2: NetworkPolicies (Zero-Trust Networking)
  ├─ Default-deny all ingress/egress
  ├─ Allow only explicitly whitelisted services
  └─ Prevent cross-tenant network traffic

Layer 3: ResourceQuotas (Resource Isolation)
  ├─ Limit CPU/Memory per tenant
  └─ Prevent "Noisy Neighbor" DoS attacks

Layer 4: Webhooks (Policy Enforcement)
  ├─ Prevent unsafe tier downgrades
  └─ Enforce immutability constraints

Layer 5: Audit Logging
  ├─ All Tenant changes recorded in etcd
  └─ Kubernetes audit logs track webhook decisions
```

### Threat Model: Compromised Tenant Pod

**Attacker Goal:** Access another tenant's data

**Attack Vector 1: NetworkPolicy Bypass**
```
Compromised Pod (tenant-acme-corp):
  ├─ Attempts: curl http://secret-service.tenant-bigbank.svc.cluster.local
  ↓
NetworkPolicy (default-deny) drops packet
  ├─ No egress rule for tenant-bigbank namespace
  ↓
Result: Request BLOCKED
```

**Attack Vector 2: RBAC Escalation**
```
Compromised Pod (tenant-acme-corp):
  ├─ Attempts: kubectl get pods -n tenant-bigbank
  │  (using acme-corp-sa token)
  ↓
Kubernetes API Server checks RBAC:
  ├─ ServiceAccount: acme-corp-sa
  ├─ Namespace: tenant-acme-corp
  ├─ Resource: pods
  ├─ Verb: get
  ├─ Result: Permission DENIED (outside namespace)
  ↓
Result: Request DENIED
```

**Attack Vector 3: Direct etcd Access**
```
Compromised Pod:
  ├─ Attempts: telnet etcd-service 2379
  ↓
NetworkPolicy blocks egress to kube-system namespace
  ├─ No route to shared infrastructure
  ↓
Result: Request BLOCKED
```

## Performance Characteristics

### Provisioning Time Targets (KPIs)

| Tier | Target | Complexity | Components Created |
|------|--------|-----------|-------------------|
| Bronze | < 1s | Low | (Stub only) |
| Silver | < 5s | Medium | Namespace, ResourceQuota, RBAC (2×), NetworkPolicy |
| Gold | < 45s | High | Silver (above) + vCluster StatefulSet + Kubeconfig Secret |

### Scalability

**Single Operator Instance Limits:**
- Max concurrent reconciliations: 3 (configurable)
- Max tenants per cluster: Limited by etcd capacity (millions)
- Typical deployment: 1-3 operator replicas for HA

**Reconciliation Performance:**
- Average Silver tier: ~2-3 seconds
- Average Gold tier: ~30-40 seconds (vCluster deployment dominant)
- Failed reconciliation: Retry after 30s (with exponential backoff)

## Extensibility Points

### Custom Tier Implementations

Future versions can add custom tiers by extending the reconciliation logic:
```golang
case platformv1alpha1.CustomTier:
    reconcileCustomTier(ctx, tenant, log)
```

### Tenant Admission Webhooks

Operators can implement additional mutating/validating webhooks:
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: custom-tenant-mutator
webhooks:
- name: custom-mutator.example.com
  clientConfig:
    service:
      name: custom-service
      namespace: default
  rules:
  - operations: [CREATE, UPDATE]
    apiGroups: [platform.io]
    resources: [tenants]
```

### Metrics & Observability Extensions

Custom metrics can be added:
```golang
customMetric := prometheus.NewGaugeVec(...)
metrics.Registry.MustRegister(customMetric)
```

## Testing Strategy

### Unit Tests
- Reconciliation logic (mocked K8s client)
- Webhook validation/mutation logic
- Helper functions (namespace naming, resource parsing)

### Integration Tests
- Real K8s cluster (Kind or Minikube)
- End-to-end tenant provisioning
- Webhook behavior
- Finalizer cleanup

### Performance Tests
- Provisioning time benchmarks
- High concurrency (100+ tenants)
- Recovery from failures

### Security Tests
- "Red Team" network isolation tests
- RBAC escape attempts
- Resource quota enforcement

## Conclusion

Tenant-Master provides a robust, secure, and scalable foundation for multi-tenant Kubernetes platforms. Its reconciliation-based design, layered security model, and flexible tier system make it suitable for diverse B2B SaaS use cases ranging from free trials to enterprise deployments.
