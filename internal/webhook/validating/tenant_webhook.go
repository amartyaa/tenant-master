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

package validating

import (
	"context"
	"fmt"
	"net/mail"

	platformv1alpha1 "github.com/amartyaa/tenant-master/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var log = logf.Log.WithName("tenant-validating-webhook")

// TenantValidatingWebhook implements the validating webhook for Tenants.
type TenantValidatingWebhook struct{}

// +kubebuilder:webhook:path=/validate-platform-io-v1alpha1-tenant,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.io,resources=tenants,verbs=create;update,versions=v1alpha1,name=vtenant.platform.io,admissionReviewVersions={v1},clientConfig={service:{name=webhook-service,namespace=system},caBundle=Cg==}

func (w *TenantValidatingWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&platformv1alpha1.Tenant{}).
		WithValidator(w).
		Complete()
}

// ValidateCreate implements the create validation logic.
func (w *TenantValidatingWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tenant, ok := obj.(*platformv1alpha1.Tenant)
	if !ok {
		return nil, nil
	}

	log.Info("validating webhook (create) called", "tenant", tenant.Name)
	return w.validateTenant(tenant)
}

// ValidateUpdate implements the update validation logic.
func (w *TenantValidatingWebhook) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	oldTenant, ok := oldObj.(*platformv1alpha1.Tenant)
	if !ok {
		return nil, nil
	}

	newTenant, ok := newObj.(*platformv1alpha1.Tenant)
	if !ok {
		return nil, nil
	}

	log.Info("validating webhook (update) called", "tenant", newTenant.Name)

	// Check for unsafe tier downgrade
	if err := w.validateTierMigration(oldTenant, newTenant); err != nil {
		return nil, err
	}

	return w.validateTenant(newTenant)
}

// ValidateDelete implements the delete validation logic (currently a no-op).
func (w *TenantValidatingWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tenant, ok := obj.(*platformv1alpha1.Tenant)
	if !ok {
		return nil, nil
	}

	log.Info("validating webhook (delete) called", "tenant", tenant.Name)
	// Allow deletion, cleanup is handled by finalizers
	return nil, nil
}

// validateTenant performs common validation on a Tenant object.
func (w *TenantValidatingWebhook) validateTenant(tenant *platformv1alpha1.Tenant) (admission.Warnings, error) {
	var allErrs field.ErrorList

	// Validate tier
	validTiers := []platformv1alpha1.TenantTier{
		platformv1alpha1.BronzeTier,
		platformv1alpha1.SilverTier,
		platformv1alpha1.GoldTier,
	}
	validTier := false
	for _, t := range validTiers {
		if tenant.Spec.Tier == t {
			validTier = true
			break
		}
	}
	if !validTier {
		allErrs = append(allErrs, field.NotSupported(
			field.NewPath("spec").Child("tier"),
			tenant.Spec.Tier,
			[]string{string(platformv1alpha1.BronzeTier), string(platformv1alpha1.SilverTier), string(platformv1alpha1.GoldTier)},
		))
	}

	// Validate owner email format
	if tenant.Spec.Owner == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec").Child("owner"),
			"owner must be specified",
		))
	} else {
		if _, err := mail.ParseAddress(tenant.Spec.Owner); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("owner"),
				tenant.Spec.Owner,
				fmt.Sprintf("invalid email format: %v", err),
			))
		}
	}

	// Validate resource quantities
	if tenant.Spec.Resources.CPU != "" {
		if _, err := parseQuantity(tenant.Spec.Resources.CPU); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("resources").Child("cpu"),
				tenant.Spec.Resources.CPU,
				fmt.Sprintf("invalid quantity: %v", err),
			))
		}
	}

	if tenant.Spec.Resources.Memory != "" {
		if _, err := parseQuantity(tenant.Spec.Resources.Memory); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("resources").Child("memory"),
				tenant.Spec.Resources.Memory,
				fmt.Sprintf("invalid quantity: %v", err),
			))
		}
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: platformv1alpha1.GroupVersion.Group, Kind: "Tenant"},
		tenant.Name,
		allErrs,
	)
}

// validateTierMigration checks for unsafe tier downgrades.
func (w *TenantValidatingWebhook) validateTierMigration(oldTenant, newTenant *platformv1alpha1.Tenant) error {
	// Define tier order (lower = less isolated)
	tierOrder := map[platformv1alpha1.TenantTier]int{
		platformv1alpha1.BronzeTier: 0,
		platformv1alpha1.SilverTier: 1,
		platformv1alpha1.GoldTier:   2,
	}

	oldOrder := tierOrder[oldTenant.Spec.Tier]
	newOrder := tierOrder[newTenant.Spec.Tier]

	// Downgrade detected (moving from higher to lower isolation)
	if newOrder < oldOrder {
		if !newTenant.Spec.AllowTierMigration {
			return apierrors.NewForbidden(
				schema.GroupResource{Group: platformv1alpha1.GroupVersion.Group, Resource: "tenants"},
				newTenant.Name,
				fmt.Errorf("unsafe tier downgrade: %s -> %s. Set spec.allowTierMigration=true to proceed (DATA MAY BE LOST)",
					oldTenant.Spec.Tier, newTenant.Spec.Tier),
			)
		}
		log.Info("tier downgrade allowed with flag", "tenant", newTenant.Name,
			"oldTier", oldTenant.Spec.Tier, "newTier", newTenant.Spec.Tier)
		// Reset the flag after migration to prevent accidental downgrades
		// newTenant.Spec.AllowTierMigration = false  // Note: Can't mutate in validator
	}

	return nil
}

// parseQuantity is a helper to parse Kubernetes resource quantities.
func parseQuantity(s string) (resource.Quantity, error) {
	if s == "" {
		return resource.Quantity{}, fmt.Errorf("empty quantity")
	}
	return resource.ParseQuantity(s)
}
