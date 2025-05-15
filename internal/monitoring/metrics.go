package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	TenantsProvisioned = prometheus.NewCounterVec( // Changed to TenantsProvisioned
		prometheus.CounterOpts{
			Name: "tenants_provisioned_total",
			Help: "Total number of tenants provisioned by status",
		},
		[]string{"status"},
	)
	ProvisioningDuration = prometheus.NewHistogram( // Changed to ProvisioningDuration
		prometheus.HistogramOpts{
			Name:    "tenant_provisioning_duration_seconds",
			Help:    "Duration of tenant provisioning in seconds",
			Buckets: prometheus.LinearBuckets(0, 1, 10), // 0 to 10 seconds
		},
	)
)

func InitMetrics() {
	err := prometheus.Register(TenantsProvisioned)
	if err != nil {
		log.Error().Err(err).Msg("Failed to register TenantsProvisioned metric")
	}

	err = prometheus.Register(ProvisioningDuration)
	if err != nil {
		log.Error().Err(err).Msg("Failed to register ProvisioningDuration metric")
	}
}
