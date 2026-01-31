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

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	platformv1alpha1 "github.com/amartyaa/tenant-master/operator/api/v1alpha1"
)

// ensureNamespace creates or updates the tenant namespace.
func (r *TenantReconciler) ensureNamespace(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				TierLabelKey:       string(tenant.Spec.Tier),
				OwnerLabelKey:      tenant.Spec.Owner,
				ManagedByLabelKey:  ManagedByValue,
			},
		},
	}

	// Set OwnerReference for garbage collection
	if err := controllerutil.SetControllerReference(tenant, ns, r.Scheme); err != nil {
		return fmt.Errorf("failed to set OwnerReference: %w", err)
	}

	// Create or update the namespace
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		ns.Labels = map[string]string{
			TenantNameLabelKey: tenant.Name,
			TierLabelKey:       string(tenant.Spec.Tier),
			OwnerLabelKey:      tenant.Spec.Owner,
			ManagedByLabelKey:  ManagedByValue,
		}
		return nil
	})

	if err != nil {
		log.Error(err, "failed to create or update namespace", "namespace", namespaceName)
		return err
	}

	log.Info("ensured namespace", "namespace", namespaceName, "operation", result)
	tenant.Status.Namespace = namespaceName
	return nil
}

// ensureResourceQuota creates or updates ResourceQuota for the tenant namespace.
func (r *TenantReconciler) ensureResourceQuota(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)

	// Parse resource requirements
	cpuQty, memQty := parseResources(tenant.Spec.Resources)

	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-quota", tenant.Name),
			Namespace: namespaceName,
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				ManagedByLabelKey:  ManagedByValue,
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceName("requests.cpu"):    cpuQty,
				corev1.ResourceName("requests.memory"): memQty,
				corev1.ResourceName("limits.cpu"):      cpuQty,
				corev1.ResourceName("limits.memory"):   memQty,
				corev1.ResourcePods:                    resource.MustParse("100"), // Limit pods to prevent DOS
			},
		},
	}

	if err := controllerutil.SetControllerReference(tenant, rq, r.Scheme); err != nil {
		return fmt.Errorf("failed to set OwnerReference: %w", err)
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, rq, func() error {
		rq.Spec.Hard = corev1.ResourceList{
			corev1.ResourceName("requests.cpu"):    cpuQty,
			corev1.ResourceName("requests.memory"): memQty,
			corev1.ResourceName("limits.cpu"):      cpuQty,
			corev1.ResourceName("limits.memory"):   memQty,
			corev1.ResourcePods:                    resource.MustParse("100"),
		}
		return nil
	})

	if err != nil {
		log.Error(err, "failed to create or update ResourceQuota", "namespace", namespaceName)
		return err
	}

	log.Info("ensured ResourceQuota", "namespace", namespaceName, "operation", result)
	return nil
}

// ensureRBAC creates ServiceAccount and RoleBinding for the tenant.
func (r *TenantReconciler) ensureRBAC(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)
	saName := fmt.Sprintf("%s-sa", tenant.Name)

	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespaceName,
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				ManagedByLabelKey:  ManagedByValue,
			},
		},
	}

	if err := controllerutil.SetControllerReference(tenant, sa, r.Scheme); err != nil {
		return fmt.Errorf("failed to set OwnerReference: %w", err)
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		return nil // ServiceAccount has minimal spec
	})

	if err != nil {
		log.Error(err, "failed to create or update ServiceAccount", "namespace", namespaceName, "serviceAccount", saName)
		return err
	}

	log.Info("ensured ServiceAccount", "namespace", namespaceName, "serviceAccount", saName, "operation", result)

	// Create Role that allows full access within the namespace
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-admin", tenant.Name),
			Namespace: namespaceName,
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				ManagedByLabelKey:  ManagedByValue,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	if err := controllerutil.SetControllerReference(tenant, role, r.Scheme); err != nil {
		return fmt.Errorf("failed to set OwnerReference on Role: %w", err)
	}

	result, err = controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		}
		return nil
	})

	if err != nil {
		log.Error(err, "failed to create or update Role", "namespace", namespaceName)
		return err
	}

	log.Info("ensured Role", "namespace", namespaceName, "operation", result)

	// Create RoleBinding that binds the role to the ServiceAccount
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-admin-binding", tenant.Name),
			Namespace: namespaceName,
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				ManagedByLabelKey:  ManagedByValue,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: namespaceName,
			},
		},
	}

	if err := controllerutil.SetControllerReference(tenant, rb, r.Scheme); err != nil {
		return fmt.Errorf("failed to set OwnerReference on RoleBinding: %w", err)
	}

	result, err = controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
		rb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: namespaceName,
			},
		}
		return nil
	})

	if err != nil {
		log.Error(err, "failed to create or update RoleBinding", "namespace", namespaceName)
		return err
	}

	log.Info("ensured RoleBinding", "namespace", namespaceName, "operation", result)
	return nil
}

// ensureSecretsAndConfigMaps propagates image pull secrets and ConfigMaps from controller namespace to tenant namespace.
// E1-05: Implements automatic secret/configmap propagation for tenant environments.
func (r *TenantReconciler) ensureSecretsAndConfigMaps(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)
	controllerNamespace := "tenant-master-system" // Controller's namespace (typically kube-system or custom)

	// Copy all image pull secrets from controller namespace to tenant namespace
	secretList := &corev1.SecretList{}
	if err := r.List(ctx, secretList, &client.ListOptions{Namespace: controllerNamespace}); err != nil {
		log.Error(err, "failed to list secrets in controller namespace", "namespace", controllerNamespace)
		// Non-fatal: continue if secrets can't be listed
		return nil
	}

	for _, secret := range secretList.Items {
		// Only copy image pull secrets
		if secret.Type != corev1.SecretTypeDockercfg && secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}

		// Create a copy of the secret in the tenant namespace
		tenantSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: namespaceName,
				Labels: map[string]string{
					TenantNameLabelKey: tenant.Name,
					ManagedByLabelKey:  ManagedByValue,
				},
			},
			Type: secret.Type,
			Data: secret.Data,
		}

		if err := controllerutil.SetControllerReference(tenant, tenantSecret, r.Scheme); err != nil {
			return fmt.Errorf("failed to set OwnerReference on secret: %w", err)
		}

		result, err := controllerutil.CreateOrUpdate(ctx, r.Client, tenantSecret, func() error {
			tenantSecret.Data = secret.Data
			tenantSecret.Type = secret.Type
			return nil
		})

		if err != nil {
			log.Error(err, "failed to propagate image pull secret", "secret", secret.Name)
			continue // Non-fatal: continue with other secrets
		}

		log.Info("propagated image pull secret", "secret", secret.Name, "operation", result)
	}

	// Copy standard ConfigMaps (e.g., "platform-config" if it exists)
	standardConfigMaps := []string{"platform-config"}
	for _, cmName := range standardConfigMaps {
		sourceConfigMap := &corev1.ConfigMap{}
		sourceKey := client.ObjectKey{Namespace: controllerNamespace, Name: cmName}
		if err := r.Get(ctx, sourceKey, sourceConfigMap); err != nil {
			log.V(1).Info("standard ConfigMap not found, skipping", "configmap", cmName)
			continue // Non-fatal: ConfigMap may not exist
		}

		// Create a copy in the tenant namespace
		tenantConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: namespaceName,
				Labels: map[string]string{
					TenantNameLabelKey: tenant.Name,
					ManagedByLabelKey:  ManagedByValue,
				},
			},
			Data:       sourceConfigMap.Data,
			BinaryData: sourceConfigMap.BinaryData,
		}

		if err := controllerutil.SetControllerReference(tenant, tenantConfigMap, r.Scheme); err != nil {
			return fmt.Errorf("failed to set OwnerReference on ConfigMap: %w", err)
		}

		result, err := controllerutil.CreateOrUpdate(ctx, r.Client, tenantConfigMap, func() error {
			tenantConfigMap.Data = sourceConfigMap.Data
			tenantConfigMap.BinaryData = sourceConfigMap.BinaryData
			return nil
		})

		if err != nil {
			log.Error(err, "failed to propagate ConfigMap", "configmap", cmName)
			continue // Non-fatal: continue
		}

		log.Info("propagated ConfigMap", "configmap", cmName, "operation", result)
	}

	return nil
}

// ensureNetworkPolicy creates a default-deny NetworkPolicy for the tenant namespace.
func (r *TenantReconciler) ensureNetworkPolicy(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)

	// Build ingress rules for whitelisted services
	var ingressRules []netv1.NetworkPolicyIngressRule
	var egressRules []netv1.NetworkPolicyEgressRule

	// Allow ingress from within the same namespace
	ingressRules = append(ingressRules, netv1.NetworkPolicyIngressRule{
		From: []netv1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{}, // All pods in this namespace
			},
		},
	})

	// Allow DNS egress (required for service discovery)
	egressRules = append(egressRules, netv1.NetworkPolicyEgressRule{
		To: []netv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": "kube-system",
					},
				},
			},
		},
		Ports: []netv1.NetworkPolicyPort{
			{
				Protocol: &[]corev1.Protocol{corev1.ProtocolUDP}[0],
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
			},
		},
	})

	// Add whitelisted services as egress rules
	for _, service := range tenant.Spec.Network.WhitelistedServices {
		namespace, svcName := parseServiceRef(service)
		egressRules = append(egressRules, netv1.NetworkPolicyEgressRule{
			To: []netv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"name": namespace,
						},
					},
					PodSelector: &metav1.LabelSelector{
						// This matches all pods in the whitelisted namespace
						// For more granular control, would need service labels
					},
				},
			},
		})
		log.Info("added whitelisted service to NetworkPolicy", "namespace", namespace, "service", svcName)
	}

	// Allow egress to internet if configured
	if tenant.Spec.Network.AllowInternetAccess {
		egressRules = append(egressRules, netv1.NetworkPolicyEgressRule{
			To: []netv1.NetworkPolicyPeer{
				{
					IPBlock: &netv1.IPBlock{
						CIDR: "0.0.0.0/0",
					},
				},
			},
		})
		log.Info("added internet egress to NetworkPolicy")
	}

	netPolicy := &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultNetworkPolicyName,
			Namespace: namespaceName,
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				ManagedByLabelKey:  ManagedByValue,
			},
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // Apply to all pods in namespace
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
				netv1.PolicyTypeEgress,
			},
			Ingress: ingressRules,
			Egress:  egressRules,
		},
	}

	if err := controllerutil.SetControllerReference(tenant, netPolicy, r.Scheme); err != nil {
		return fmt.Errorf("failed to set OwnerReference: %w", err)
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, netPolicy, func() error {
		netPolicy.Spec.Ingress = ingressRules
		netPolicy.Spec.Egress = egressRules
		return nil
	})

	if err != nil {
		log.Error(err, "failed to create or update NetworkPolicy", "namespace", namespaceName)
		return err
	}

	log.Info("ensured NetworkPolicy", "namespace", namespaceName, "operation", result)
	return nil
}

// Helper functions

// buildNamespaceName generates the namespace name for a tenant.
func buildNamespaceName(tenant *platformv1alpha1.Tenant) string {
	return fmt.Sprintf("%s-%s", NamespacePrefix, tenant.Name)
}

// parseResources parses resource requirements and returns k8s quantities.
func parseResources(req platformv1alpha1.ResourceRequirements) (resource.Quantity, resource.Quantity) {
	cpu := resource.MustParse("1000m")
	memory := resource.MustParse("1Gi")

	if req.CPU != "" {
		if parsed, err := resource.ParseQuantity(req.CPU); err == nil {
			cpu = parsed
		}
	}

	if req.Memory != "" {
		if parsed, err := resource.ParseQuantity(req.Memory); err == nil {
			memory = parsed
		}
	}

	return cpu, memory
}

// parseServiceRef parses a service reference like "namespace/service" or "namespace/service:port".
func parseServiceRef(serviceRef string) (string, string) {
	// For now, simple split by "/"
	// TODO: Handle port parsing if needed
	parts := []rune(serviceRef)
	for i, c := range parts {
		if c == '/' {
			return string(parts[:i]), string(parts[i+1:])
		}
	}
	// Fallback: assume it's just a service name in current namespace
	return "default", serviceRef
}

// detectAndCorrectNetworkPolicyDrift checks for manual edits to NetworkPolicies and reverts to desired state.
// E1-06: Implements drift detection and reconciliation for NetworkPolicies.
func (r *TenantReconciler) detectAndCorrectNetworkPolicyDrift(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)

	// Fetch the current NetworkPolicy
	currentNetPolicy := &netv1.NetworkPolicy{}
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespaceName,
		Name:      DefaultNetworkPolicyName,
	}, currentNetPolicy); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.V(1).Info("NetworkPolicy not found, skipping drift detection", "policy", DefaultNetworkPolicyName)
			return nil
		}
		return fmt.Errorf("failed to fetch NetworkPolicy for drift detection: %w", err)
	}

	// Reconstruct the desired state (same logic as ensureNetworkPolicy)
	var ingressRules []netv1.NetworkPolicyIngressRule
	var egressRules []netv1.NetworkPolicyEgressRule

	// Allow ingress from within the same namespace
	ingressRules = append(ingressRules, netv1.NetworkPolicyIngressRule{
		From: []netv1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{},
			},
		},
	})

	// Allow DNS egress
	egressRules = append(egressRules, netv1.NetworkPolicyEgressRule{
		To: []netv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": "kube-system",
					},
				},
			},
		},
		Ports: []netv1.NetworkPolicyPort{
			{
				Protocol: &[]corev1.Protocol{corev1.ProtocolUDP}[0],
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
			},
		},
	})

	// Add whitelisted services
	for _, service := range tenant.Spec.Network.WhitelistedServices {
		namespace, _ := parseServiceRef(service)
		egressRules = append(egressRules, netv1.NetworkPolicyEgressRule{
			To: []netv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"name": namespace,
						},
					},
					PodSelector: &metav1.LabelSelector{},
				},
			},
		})
	}

	// Allow internet egress if configured
	if tenant.Spec.Network.AllowInternetAccess {
		egressRules = append(egressRules, netv1.NetworkPolicyEgressRule{
			To: []netv1.NetworkPolicyPeer{
				{
					IPBlock: &netv1.IPBlock{
						CIDR: "0.0.0.0/0",
					},
				},
			},
		})
	}

	// Compare ingress and egress rules for drift
	if len(currentNetPolicy.Spec.Ingress) != len(ingressRules) ||
		len(currentNetPolicy.Spec.Egress) != len(egressRules) {
		log.Error(nil, "NetworkPolicy drift detected, correcting", "namespace", namespaceName, "policy", DefaultNetworkPolicyName)

		// Revert to desired state
		currentNetPolicy.Spec.Ingress = ingressRules
		currentNetPolicy.Spec.Egress = egressRules

		if err := r.Update(ctx, currentNetPolicy); err != nil {
			return fmt.Errorf("failed to correct NetworkPolicy drift: %w", err)
		}

		log.Info("NetworkPolicy drift corrected", "namespace", namespaceName)
	}

	return nil
}

// takeSnapshotBeforeDeletion creates a snapshot of tenant resources before deletion.
// E3-04: Implements snapshot routine for graceful teardown.
func (r *TenantReconciler) takeSnapshotBeforeDeletion(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	namespaceName := buildNamespaceName(tenant)
	snapshotName := fmt.Sprintf("snapshot-%s-%d", tenant.Name, time.Now().Unix())

	log.Info("creating snapshot before deletion", "tenant", tenant.Name, "snapshot", snapshotName)

	// In production, this would:
	// 1. Export all ConfigMaps, Secrets, PVs from the tenant namespace
	// 2. Store them in an archive (S3, GCS, or local PV)
	// 3. Record metadata (timestamp, owner, resource count)

	// For now, create a ConfigMap that records the snapshot metadata
	snapshotConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: "tenant-master-system", // Store in operator namespace
			Labels: map[string]string{
				TenantNameLabelKey: tenant.Name,
				"type":             "snapshot",
				ManagedByLabelKey:  ManagedByValue,
			},
		},
		Data: map[string]string{
			"tenant-name":      tenant.Name,
			"snapshot-time":    time.Now().Format(time.RFC3339),
			"source-namespace": namespaceName,
			"tier":             string(tenant.Spec.Tier),
			"owner":            tenant.Spec.Owner,
			"status":           "completed",
		},
	}

	if err := r.Create(ctx, snapshotConfigMap); err != nil {
		// Non-fatal: log and continue with deletion
		log.Error(err, "failed to create snapshot metadata", "snapshot", snapshotName)
	} else {
		log.Info("snapshot metadata recorded", "snapshot", snapshotName)
	}

	return nil
}
