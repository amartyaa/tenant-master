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
)

func init() {
	// Register metrics
	metrics.Registry.MustRegister(ProvisioningTimeHistogram)
	metrics.Registry.MustRegister(ActiveTenantsGauge)
	metrics.Registry.MustRegister(ReconciliationErrors)
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
