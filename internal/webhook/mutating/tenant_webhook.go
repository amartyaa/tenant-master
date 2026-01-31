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

package mutating

import (
	"context"
	"strings"

	platformv1alpha1 "github.com/amartyaa/tenant-master/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("tenant-mutating-webhook")

// TenantMutatingWebhook implements the mutating webhook for Tenants.
type TenantMutatingWebhook struct{}

// +kubebuilder:webhook:path=/mutate-platform-io-v1alpha1-tenant,mutating=true,failurePolicy=fail,sideEffects=None,groups=platform.io,resources=tenants,verbs=create;update,versions=v1alpha1,name=mtenant.platform.io,admissionReviewVersions={v1},clientConfig={service:{name=webhook-service,namespace=system},caBundle=Cg==}

func (w *TenantMutatingWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&platformv1alpha1.Tenant{}).
		WithDefaulter(w).
		Complete()
}

// Default implements the default mutation logic.
func (w *TenantMutatingWebhook) Default(ctx context.Context, obj runtime.Object) error {
	tenant, ok := obj.(*platformv1alpha1.Tenant)
	if !ok {
		return nil
	}

	log.Info("mutating webhook called", "tenant", tenant.Name)

	// Default tier to Silver if not specified
	if tenant.Spec.Tier == "" {
		log.Info("defaulting tier to Silver", "tenant", tenant.Name)
		tenant.Spec.Tier = platformv1alpha1.SilverTier
	}

	// Normalize owner email to lowercase
	if tenant.Spec.Owner != "" {
		tenant.Spec.Owner = strings.ToLower(tenant.Spec.Owner)
	}

	// Set default resources if not specified
	if tenant.Spec.Resources.CPU == "" {
		tenant.Spec.Resources.CPU = "1000m"
	}
	if tenant.Spec.Resources.Memory == "" {
		tenant.Spec.Resources.Memory = "1Gi"
	}

	// Set default network config
	if len(tenant.Spec.Network.WhitelistedServices) == 0 {
		// Default: allow DNS for service discovery
		tenant.Spec.Network.WhitelistedServices = []string{}
	}

	log.Info("mutating webhook completed", "tenant", tenant.Name, "tier", tenant.Spec.Tier)
	return nil
}
