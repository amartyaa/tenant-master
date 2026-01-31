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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantTier represents the isolation level for a tenant.
// +kubebuilder:validation:Enum=Bronze;Silver;Gold
type TenantTier string

const (
	// BronzeTier: Soft isolation using application-level logic.
	// Shared resources, minimal security enforcement.
	BronzeTier TenantTier = "Bronze"

	// SilverTier: Hard isolation via dedicated namespace.
	// Includes ResourceQuotas, NetworkPolicies, and RBAC.
	SilverTier TenantTier = "Silver"

	// GoldTier: Extreme isolation via dedicated vCluster.
	// Tenant has their own Kubernetes API server and full admin privileges.
	GoldTier TenantTier = "Gold"
)

// TenantState represents the reconciliation state of a tenant.
// +kubebuilder:validation:Enum=Provisioning;Ready;Failed;Suspended;Terminating
type TenantState string

const (
	// StateProvisioning: Tenant resources are being created.
	StateProvisioning TenantState = "Provisioning"

	// StateReady: Tenant is fully provisioned and ready for use.
	StateReady TenantState = "Ready"

	// StateFailed: Tenant provisioning failed. Check events for details.
	StateFailed TenantState = "Failed"

	// StateSuspended: Tenant is in sleep mode (scale-to-zero).
	StateSuspended TenantState = "Suspended"

	// StateTerminating: Tenant is being deleted.
	StateTerminating TenantState = "Terminating"
)

// ResourceRequirements defines CPU, memory, and storage constraints for a tenant.
type ResourceRequirements struct {
	// CPU request/limit in millicores (e.g., "4000m").
	// +kubebuilder:validation:Pattern=^(\d+m|\d+\.?\d*|\d*\.?\d+)$
	CPU string `json:"cpu,omitempty"`

	// Memory request/limit (e.g., "8Gi", "1024Mi").
	// +kubebuilder:validation:Pattern=^(\d+Mi|\d+Gi|\d+Ti)$
	Memory string `json:"memory,omitempty"`

	// StorageClass name for PersistentVolumeClaims (e.g., "fast-ssd", "standard").
	StorageClass string `json:"storageClass,omitempty"`
}

// NetworkConfig defines network isolation and egress rules for a tenant.
type NetworkConfig struct {
	// AllowInternetAccess determines if the tenant can reach external IPs.
	// Default: false (zero-trust networking).
	AllowInternetAccess bool `json:"allowInternetAccess,omitempty"`

	// WhitelistedServices is a list of allowed egress destinations.
	// Format: "namespace/service" or "namespace/service:port".
	// Example: ["shared-services/auth-api", "monitoring/prometheus:9090"]
	WhitelistedServices []string `json:"whitelistedServices,omitempty"`
}

// TenantSpec defines the desired state of a Tenant.
type TenantSpec struct {
	// Tier defines the isolation level for this tenant.
	// +kubebuilder:validation:Required
	Tier TenantTier `json:"tier"`

	// Owner is the email/identifier of the tenant owner for notifications.
	// +kubebuilder:validation:MinLength=1
	Owner string `json:"owner"`

	// Resources defines CPU, memory, and storage constraints.
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Network defines network policies and egress rules.
	Network NetworkConfig `json:"network,omitempty"`

	// AllowTierMigration is a flag to allow unsafe downgrades (e.g., Gold -> Bronze).
	// Must be explicitly set to true. Used for data migration workflows.
	AllowTierMigration bool `json:"allowTierMigration,omitempty"`

	// Suspend can be set to true to scale the tenant to zero replicas (cost savings).
	Suspend bool `json:"suspend,omitempty"`
}

// TenantStatus defines the observed state of a Tenant.
type TenantStatus struct {
	// State represents the current provisioning state of the tenant.
	State TenantState `json:"state,omitempty"`

	// Namespace is the name of the Kubernetes namespace allocated to this tenant.
	Namespace string `json:"namespace,omitempty"`

	// APIEndpoint is the connection address for Gold tier vClusters.
	// Format: "https://acme.k8s.myplatform.com"
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// AdminKubeconfigSecret is the name of the Secret containing the kubeconfig for Gold tier.
	// Populated only for Gold tier tenants.
	AdminKubeconfigSecret string `json:"adminKubeconfigSecret,omitempty"`

	// ProvisioningStartTime records when provisioning began.
	ProvisioningStartTime *metav1.Time `json:"provisioningStartTime,omitempty"`

	// LastUpdateTime records the last successful reconciliation.
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastError records the last error encountered during reconciliation.
	LastError string `json:"lastError,omitempty"`

	// ObservedGeneration reflects the generation of the Spec that was last reconciled.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// Tenant is the Schema for the tenants API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ten;plural=tenants
// +kubebuilder:printcolumn:name="Tier",type=string,JSONPath=`.spec.tier`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// TenantList contains a list of Tenant objects.
// +kubebuilder:object:root=true
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}

// DeepCopyInto for nested types not generated by controller-gen in this layout.
// These helpers ensure proper deep copies for slices and pointer fields.
func (in *ResourceRequirements) DeepCopyInto(out *ResourceRequirements) {
	*out = *in
}

func (in *ResourceRequirements) DeepCopy() *ResourceRequirements {
	if in == nil {
		return nil
	}
	out := new(ResourceRequirements)
	in.DeepCopyInto(out)
	return out
}

func (in *NetworkConfig) DeepCopyInto(out *NetworkConfig) {
	*out = *in
	if in.WhitelistedServices != nil {
		out.WhitelistedServices = make([]string, len(in.WhitelistedServices))
		copy(out.WhitelistedServices, in.WhitelistedServices)
	}
}

func (in *NetworkConfig) DeepCopy() *NetworkConfig {
	if in == nil {
		return nil
	}
	out := new(NetworkConfig)
	in.DeepCopyInto(out)
	return out
}

func (in *TenantSpec) DeepCopyInto(out *TenantSpec) {
	*out = *in
	// Deep copy nested structs
	in.Resources.DeepCopyInto(&out.Resources)
	in.Network.DeepCopyInto(&out.Network)
}

func (in *TenantSpec) DeepCopy() *TenantSpec {
	if in == nil {
		return nil
	}
	out := new(TenantSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *TenantStatus) DeepCopyInto(out *TenantStatus) {
	*out = *in
	if in.ProvisioningStartTime != nil {
		out.ProvisioningStartTime = in.ProvisioningStartTime.DeepCopy()
	}
	if in.LastUpdateTime != nil {
		out.LastUpdateTime = in.LastUpdateTime.DeepCopy()
	}
}

func (in *TenantStatus) DeepCopy() *TenantStatus {
	if in == nil {
		return nil
	}
	out := new(TenantStatus)
	in.DeepCopyInto(out)
	return out
}
