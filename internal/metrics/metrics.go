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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ProvisioningTimeHistogram measures the time taken to provision a tenant.
	ProvisioningTimeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tenant_provisioning_seconds",
			Help:    "Time taken to provision a tenant in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s, 2s, 4s, 8s, ..., 512s
		},
		[]string{"tier"},
	)

	// ActiveTenantsGauge tracks the number of active tenants per tier.
	ActiveTenantsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_tenants_count",
			Help: "Number of active tenants by tier",
		},
		[]string{"tier"},
	)

	// ReconciliationErrors tracks reconciliation failures.
	ReconciliationErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "reconciliation_errors_total",
			Help: "Total number of reconciliation errors",
		},
	)

	// E3-03: Enhanced metrics for Phase 2
	// ReconciliationDurationHistogram measures reconciliation loop duration.
	ReconciliationDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "reconciliation_duration_seconds",
			Help:    "Duration of reconciliation loop in seconds",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 50), // 0.1s to 5.1s in 0.1s increments
		},
		[]string{"tier", "operation"},
	)

	// ResourceUtilizationGauge tracks resource utilization per tenant.
	ResourceUtilizationGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenant_resource_utilization",
			Help: "Current resource utilization of a tenant (CPU, Memory)",
		},
		[]string{"tenant", "tier", "resource_type"},
	)

	// ErrorRateByTierCounter tracks error rates per tier.
	ErrorRateByTierCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reconciliation_errors_by_tier_total",
			Help: "Total reconciliation errors per tier",
		},
		[]string{"tier", "error_type"},
	)

	// NetworkPolicyDriftDetectedCounter tracks detected NetworkPolicy drifts.
	NetworkPolicyDriftDetectedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "network_policy_drift_detected_total",
			Help: "Total times NetworkPolicy drift was detected and corrected",
		},
		[]string{"tenant", "namespace"},
	)
)

func init() {
	// Register metrics
	metrics.Registry.MustRegister(ProvisioningTimeHistogram)
	metrics.Registry.MustRegister(ActiveTenantsGauge)
	metrics.Registry.MustRegister(ReconciliationErrors)

	// E3-03: Register enhanced metrics
	metrics.Registry.MustRegister(ReconciliationDurationHistogram)
	metrics.Registry.MustRegister(ResourceUtilizationGauge)
	metrics.Registry.MustRegister(ErrorRateByTierCounter)
	metrics.Registry.MustRegister(NetworkPolicyDriftDetectedCounter)
}

// RecordProvisioningTime records the provisioning time for a tenant.
func RecordProvisioningTime(tier string, seconds float64) {
	ProvisioningTimeHistogram.WithLabelValues(tier).Observe(seconds)
}

// RecordActiveTenant increments the active tenant count for a tier.
func RecordActiveTenant(tier string) {
	ActiveTenantsGauge.WithLabelValues(tier).Inc()
}

// DecrementActiveTenant decrements the active tenant count for a tier.
func DecrementActiveTenant(tier string) {
	ActiveTenantsGauge.WithLabelValues(tier).Dec()
}

// E3-03: Enhanced metric recording functions
// RecordReconciliationDuration records the duration of a reconciliation operation.
func RecordReconciliationDuration(tier, operation string, seconds float64) {
	ReconciliationDurationHistogram.WithLabelValues(tier, operation).Observe(seconds)
}

// RecordResourceUtilization records the current resource utilization of a tenant.
func RecordResourceUtilization(tenant, tier, resourceType string, value float64) {
	ResourceUtilizationGauge.WithLabelValues(tenant, tier, resourceType).Set(value)
}

// RecordErrorByTier records an error for a specific tier.
func RecordErrorByTier(tier, errorType string) {
	ErrorRateByTierCounter.WithLabelValues(tier, errorType).Inc()
}

// RecordNetworkPolicyDriftDetected records when NetworkPolicy drift is detected.
func RecordNetworkPolicyDriftDetected(tenant, namespace string) {
	NetworkPolicyDriftDetectedCounter.WithLabelValues(tenant, namespace).Inc()
}
