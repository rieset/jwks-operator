package config

import (
	"time"
)

// DefaultConfig returns a Config with default values
// Note: Namespace must be set from Kubernetes environment
func DefaultConfig() *Config {
	return &Config{
		Namespace:                "", // Must be set from Kubernetes environment
		ReconcileInterval:        Duration{Duration: 5 * time.Minute},
		JWKSUpdateInterval:       Duration{Duration: 6 * time.Hour},
		JWKSVerificationInterval: Duration{Duration: 1 * time.Minute},
		MaxOldKeys:               3,
		DefaultOldKeysTTL:        Duration{Duration: 720 * time.Hour}, // 30 days
		DefaultUpdateStrategy:    "rolling",
		DefaultKeepOldKeys:       true,
		CleanupOnDelete:          false,
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			VerboseReconcile: false,
		},
		Metrics: MetricsConfig{
			Port:     8080,
			Path:     "/metrics",
			Detailed: true,
		},
		Health: HealthConfig{
			Port:          8081,
			LivenessPath:  "/healthz",
			ReadinessPath: "/readyz",
		},
		Retry: RetryConfig{
			MaxAttempts:       5,
			InitialDelay:      Duration{Duration: 5 * time.Second},
			MaxDelay:          Duration{Duration: 5 * time.Minute},
			BackoffMultiplier: 2.0,
		},
		Cache: CacheConfig{
			EnableJWKSCache: true,
			JWKSCacheTTL:    Duration{Duration: 1 * time.Hour},
			EnableCertCache: true,
			CertCacheTTL:    Duration{Duration: 30 * time.Minute},
		},
		Validation: ValidationConfig{
			ValidateCertificates: true,
			ValidateJWKS:         true,
			ValidateJSON:         true,
		},
		Security: SecurityConfig{
			MinimalRBAC:          true,
			CheckOwnerReferences: true,
			UseServiceAccount:    true,
		},
		Nginx: NginxConfig{
			Image:       DefaultNginxImage,
			Port:        DefaultNginxPort,
			Replicas:    DefaultNginxReplicas,
			CacheMaxAge: DefaultCacheMaxAge,
			Resources: NginxResources{
				Requests: NginxResourceRequirements{
					CPU:    DefaultNginxCPURequest,
					Memory: DefaultNginxMemoryRequest,
				},
				Limits: NginxResourceRequirements{
					CPU:    DefaultNginxCPULimit,
					Memory: DefaultNginxMemoryLimit,
				},
			},
		},
		Verification: VerificationConfig{
			Timeout:        DefaultVerificationTimeout,
			RetryCount:     DefaultVerificationRetryCount,
			RetryDelay:     DefaultVerificationRetryDelay,
			ContextTimeout: DefaultVerificationContextTimeout,
		},
	}
}

// DefaultReconcileInterval returns the default reconcile interval
func DefaultReconcileInterval() Duration {
	return Duration{Duration: 5 * time.Minute}
}

// DefaultJWKSUpdateInterval returns the default JWKS update interval
func DefaultJWKSUpdateInterval() Duration {
	return Duration{Duration: 6 * time.Hour}
}

// DefaultOldKeysTTL returns the default TTL for old keys
func DefaultOldKeysTTL() Duration {
	return Duration{Duration: 720 * time.Hour} // 30 days
}
