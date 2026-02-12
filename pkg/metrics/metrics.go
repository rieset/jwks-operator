package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// ResultSuccess indicates a successful operation
	ResultSuccess = "success"
	// ResultError indicates a failed operation
	ResultError = "error"
)

var (
	// ReconcileTotal is a counter for total number of reconciliations
	ReconcileTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_operator_reconcile_total",
			Help: "Total number of reconciliations",
		},
		[]string{"result"}, // result: success, error
	)

	// ReconcileDuration is a histogram for reconciliation duration
	ReconcileDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "jwks_operator_reconcile_duration_seconds",
			Help:    "Duration of reconciliation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"}, // result: success, error
	)

	// ConfigMapUpdatesTotal is a counter for ConfigMap updates
	ConfigMapUpdatesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_operator_configmap_updates_total",
			Help: "Total number of ConfigMap updates",
		},
		[]string{"type", "result"}, // type: jwks, nginx, result: success, error
	)

	// JWKSGenerationTotal is a counter for JWKS generation attempts
	JWKSGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_operator_jwks_generation_total",
			Help: "Total number of JWKS generation attempts",
		},
		[]string{"result"}, // result: success, error
	)

	// NginxOperationsTotal is a counter for nginx operations
	NginxOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_operator_nginx_operations_total",
			Help: "Total number of nginx operations",
		},
		[]string{"operation", "result"}, // operation: deployment, service, config, result: success, error
	)

	// JWKSVerificationTotal is a counter for JWKS verification attempts
	JWKSVerificationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_operator_jwks_verification_total",
			Help: "Total number of JWKS verification attempts",
		},
		[]string{"result"}, // result: success, error
	)

	// ErrorsTotal is a counter for errors by type
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_operator_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type"}, // type: secret_not_found, jwks_generation_failed, configmap_update_failed, etc.
	)
)

// RecordReconcile records a reconciliation attempt
func RecordReconcile(result string, duration float64) {
	ReconcileTotal.WithLabelValues(result).Inc()
	ReconcileDuration.WithLabelValues(result).Observe(duration)
}

// RecordConfigMapUpdate records a ConfigMap update
func RecordConfigMapUpdate(configMapType, result string) {
	ConfigMapUpdatesTotal.WithLabelValues(configMapType, result).Inc()
}

// RecordJWKSGeneration records a JWKS generation attempt
func RecordJWKSGeneration(result string) {
	JWKSGenerationTotal.WithLabelValues(result).Inc()
}

// RecordNginxOperation records a nginx operation
func RecordNginxOperation(operation, result string) {
	NginxOperationsTotal.WithLabelValues(operation, result).Inc()
}

// RecordJWKSVerification records a JWKS verification attempt
func RecordJWKSVerification(result string) {
	JWKSVerificationTotal.WithLabelValues(result).Inc()
}

// RecordError records an error by type
func RecordError(errorType string) {
	ErrorsTotal.WithLabelValues(errorType).Inc()
}
