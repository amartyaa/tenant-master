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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
