/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

const (
	// TenantFinalizerName is the finalizer used for cleanup on Tenant deletion.
	TenantFinalizerName = "tenant.platform.io/finalizer"

	// NamespacePrefix is the prefix for tenant namespaces.
	NamespacePrefix = "tenant"

	// OwnerLabelKey is the label key for tenant owner.
	OwnerLabelKey = "tenant.platform.io/owner"

	// TierLabelKey is the label key for tenant tier.
	TierLabelKey = "tenant.platform.io/tier"

	// TenantNameLabelKey is the label key for tenant name.
	TenantNameLabelKey = "tenant.platform.io/name"

	// ManagedByLabelKey indicates the resource is managed by Tenant-Master.
	ManagedByLabelKey = "app.kubernetes.io/managed-by"
	ManagedByValue    = "tenant-master"

	// DefaultNetworkPolicyName is the name of the default-deny NetworkPolicy.
	DefaultNetworkPolicyName = "default-deny-all"

	// VClusterReleaseName is the Helm release name for vCluster deployments.
	VClusterReleaseName = "vcluster"

	// KubeconfigSecretSuffix is the suffix for kubeconfig secrets.
	KubeconfigSecretSuffix = "kubeconfig"
)

// ErrorReasonTimeout indicates a reconciliation timeout.
const ErrorReasonTimeout = "Timeout"

// ErrorReasonNamespaceCreation indicates namespace creation failure.
const ErrorReasonNamespaceCreation = "NamespaceCreationFailed"

// ErrorReasonResourceQuotaCreation indicates ResourceQuota creation failure.
const ErrorReasonResourceQuotaCreation = "ResourceQuotaCreationFailed"

// ErrorReasonNetworkPolicyCreation indicates NetworkPolicy creation failure.
const ErrorReasonNetworkPolicyCreation = "NetworkPolicyCreationFailed"

// ErrorReasonRBACCreation indicates RBAC creation failure.
const ErrorReasonRBACCreation = "RBACCreationFailed"

// ErrorReasonVClusterDeployment indicates vCluster deployment failure.
const ErrorReasonVClusterDeployment = "VClusterDeploymentFailed"

// ErrorReasonKubeconfigRetrieval indicates kubeconfig retrieval failure.
const ErrorReasonKubeconfigRetrieval = "KubeconfigRetrievalFailed"
