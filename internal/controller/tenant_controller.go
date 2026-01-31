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
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/amartyaa/tenant-master/operator/api/v1alpha1"
	"github.com/amartyaa/tenant-master/operator/internal/metrics"
)

// TenantReconciler reconciles a Tenant object.
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=platform.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.io,resources=tenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.io,resources=tenants/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile implements the reconciliation loop for a Tenant.
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("tenant", req.NamespacedName)

	// Fetch the Tenant object
	tenant := &platformv1alpha1.Tenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Record start time for metrics
	startTime := time.Now()

	// Handle deletion
	if !tenant.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, tenant, log)
	}

	// Ensure finalizer is set
	if !controllerutil.ContainsFinalizer(tenant, TenantFinalizerName) {
		controllerutil.AddFinalizer(tenant, TenantFinalizerName)
		if err := r.Update(ctx, tenant); err != nil {
			log.Error(err, "failed to add finalizer")
			metrics.ReconciliationErrors.Inc()
			return ctrl.Result{}, err
		}
	}

	// Update status to Provisioning if not yet started
	if tenant.Status.State == "" {
		tenant.Status.State = platformv1alpha1.StateProvisioning
		tenant.Status.ProvisioningStartTime = &metav1.Time{Time: time.Now()}
		if err := r.Status().Update(ctx, tenant); err != nil {
			log.Error(err, "failed to update status to Provisioning")
			metrics.ReconciliationErrors.Inc()
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Main reconciliation logic based on tier
	var reconcileErr error
	switch tenant.Spec.Tier {
	case platformv1alpha1.SilverTier:
		reconcileErr = r.reconcileSilverTier(ctx, tenant, log)
	case platformv1alpha1.GoldTier:
		reconcileErr = r.reconcileGoldTier(ctx, tenant, log)
	case platformv1alpha1.BronzeTier:
		// Bronze tier: minimal isolation, placeholder for future implementation
		log.Info("Bronze tier provisioning (minimal isolation)", "tenant", tenant.Name)
		tenant.Status.State = platformv1alpha1.StateReady
	default:
		reconcileErr = fmt.Errorf("unknown tier: %s", tenant.Spec.Tier)
	}

	// Record provisioning time metric
	provisioningTime := time.Since(startTime).Seconds()
	metrics.RecordProvisioningTime(string(tenant.Spec.Tier), provisioningTime)

	// Update status based on reconciliation result
	if reconcileErr != nil {
		log.Error(reconcileErr, "reconciliation failed")
		tenant.Status.State = platformv1alpha1.StateFailed
		tenant.Status.LastError = reconcileErr.Error()
		metrics.ReconciliationErrors.Inc()
		if err := r.Status().Update(ctx, tenant); err != nil {
			log.Error(err, "failed to update status to Failed")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, reconcileErr
	}

	// Update last update time and observed generation
	tenant.Status.LastUpdateTime = &metav1.Time{Time: time.Now()}
	tenant.Status.ObservedGeneration = tenant.Generation
	if err := r.Status().Update(ctx, tenant); err != nil {
		log.Error(err, "failed to update status")
		metrics.ReconciliationErrors.Inc()
		return ctrl.Result{Requeue: true}, err
	}

	metrics.RecordActiveTenant(string(tenant.Spec.Tier))
	log.Info("reconciliation completed successfully", "state", tenant.Status.State)
	return ctrl.Result{}, nil
}

// reconcileSilverTier handles the Silver tier provisioning (namespace-isolated).
func (r *TenantReconciler) reconcileSilverTier(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	// Create namespace
	if err := r.ensureNamespace(ctx, tenant, log); err != nil {
		return fmt.Errorf("namespace creation failed: %w", err)
	}

	// Create ResourceQuota
	if err := r.ensureResourceQuota(ctx, tenant, log); err != nil {
		return fmt.Errorf("resource quota creation failed: %w", err)
	}

	// Create RBAC (ServiceAccount + RoleBinding)
	if err := r.ensureRBAC(ctx, tenant, log); err != nil {
		return fmt.Errorf("RBAC creation failed: %w", err)
	}

	// Create default-deny NetworkPolicy
	if err := r.ensureNetworkPolicy(ctx, tenant, log); err != nil {
		return fmt.Errorf("network policy creation failed: %w", err)
	}

	tenant.Status.State = platformv1alpha1.StateReady
	return nil
}

// reconcileGoldTier handles the Gold tier provisioning (vCluster-isolated).
func (r *TenantReconciler) reconcileGoldTier(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) error {
	// First, ensure the base namespace and policies are set up
	if err := r.reconcileSilverTier(ctx, tenant, log); err != nil {
		return fmt.Errorf("failed to set up base Silver tier resources: %w", err)
	}

	// Deploy vCluster via Helm
	if err := r.ensureVCluster(ctx, tenant, log); err != nil {
		return fmt.Errorf("vCluster deployment failed: %w", err)
	}

	// Retrieve and store kubeconfig
	if err := r.ensureKubeconfigSecret(ctx, tenant, log); err != nil {
		return fmt.Errorf("kubeconfig retrieval failed: %w", err)
	}

	tenant.Status.State = platformv1alpha1.StateReady
	return nil
}

// handleDeletion handles the Tenant deletion lifecycle (finalizers).
func (r *TenantReconciler) handleDeletion(ctx context.Context, tenant *platformv1alpha1.Tenant, log logr.Logger) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(tenant, TenantFinalizerName) {
		tenant.Status.State = platformv1alpha1.StateTerminating
		if err := r.Status().Update(ctx, tenant); err != nil {
			log.Error(err, "failed to update status to Terminating")
		}

		// Execute cleanup logic (namespace deletion is handled by OwnerReferences)
		log.Info("cleaning up tenant resources", "tenant", tenant.Name)

		// Remove finalizer
		controllerutil.RemoveFinalizer(tenant, TenantFinalizerName)
		if err := r.Update(ctx, tenant); err != nil {
			log.Error(err, "failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Tenant{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3,
		}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Only reconcile if spec or deletion timestamp changed
				oldTenant := e.ObjectOld.(*platformv1alpha1.Tenant)
				newTenant := e.ObjectNew.(*platformv1alpha1.Tenant)

				specChanged := !reflect.DeepEqual(oldTenant.Spec, newTenant.Spec)

				deletionChanged := false
				if oldTenant.DeletionTimestamp == nil && newTenant.DeletionTimestamp != nil {
					deletionChanged = true
				} else if oldTenant.DeletionTimestamp != nil && newTenant.DeletionTimestamp == nil {
					deletionChanged = true
				} else if oldTenant.DeletionTimestamp != nil && newTenant.DeletionTimestamp != nil {
					if !oldTenant.DeletionTimestamp.Time.Equal(newTenant.DeletionTimestamp.Time) {
						deletionChanged = true
					}
				}

				return specChanged || deletionChanged
			},
		}).
		Complete(r)
}
