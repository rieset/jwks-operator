package config

import "time"

// Nginx constants
const (
	// DefaultNginxImage is the default nginx container image
	DefaultNginxImage = "nginx:1.25-alpine"
	// DefaultNginxPort is the default nginx port
	DefaultNginxPort = 80
	// DefaultNginxReplicas is the default number of nginx replicas
	DefaultNginxReplicas = 1
)

// Verification constants
const (
	// DefaultVerificationTimeout is the default timeout for JWKS verification
	DefaultVerificationTimeout = 10 * time.Second
	// DefaultVerificationRetryCount is the default number of retry attempts for verification
	DefaultVerificationRetryCount = 3
	// DefaultVerificationRetryDelay is the default delay between retry attempts
	DefaultVerificationRetryDelay = 2 * time.Second
	// DefaultVerificationContextTimeout is the default context timeout for verification
	DefaultVerificationContextTimeout = 30 * time.Second
)

// Cache constants
const (
	// DefaultCacheMaxAge is the default cache max-age in seconds (1 hour)
	DefaultCacheMaxAge = 3600
)

// Resource constants
const (
	// DefaultNginxCPURequest is the default CPU request for nginx
	DefaultNginxCPURequest = "50m"
	// DefaultNginxMemoryRequest is the default memory request for nginx
	DefaultNginxMemoryRequest = "64Mi"
	// DefaultNginxCPULimit is the default CPU limit for nginx
	DefaultNginxCPULimit = "200m"
	// DefaultNginxMemoryLimit is the default memory limit for nginx
	DefaultNginxMemoryLimit = "128Mi"
)

// Label constants
const (
	// LabelApp is the app label key
	LabelApp = "app"
	// LabelJWKSConfig is the jwks-config label key
	LabelJWKSConfig = "jwks-config"
	// LabelManagedBy is the managed-by label key
	LabelManagedBy = "managed-by"
	// LabelManagedByValue is the managed-by label value
	LabelManagedByValue = "jwks-operator"
	// LabelAppValue is the app label value for nginx
	LabelAppValue = "nginx-jwks"
)

// Probe constants
const (
	// DefaultLivenessProbeInitialDelay is the default initial delay for liveness probe
	DefaultLivenessProbeInitialDelay = 10
	// DefaultLivenessProbePeriod is the default period for liveness probe
	DefaultLivenessProbePeriod = 10
	// DefaultReadinessProbeInitialDelay is the default initial delay for readiness probe
	DefaultReadinessProbeInitialDelay = 5
	// DefaultReadinessProbePeriod is the default period for readiness probe
	DefaultReadinessProbePeriod = 5
)

// Health check paths
const (
	// HealthCheckPath is the path for health check
	HealthCheckPath = "/healthz"
	// JWKSEndpointPath is the path for JWKS endpoint
	JWKSEndpointPath = "/jwks.json"
)

// ConfigMap keys
const (
	// ConfigMapKeyJWKS is the key for JWKS data in ConfigMap
	ConfigMapKeyJWKS = "jwks.json"
	// ConfigMapKeyNginxConfig is the key for nginx config in ConfigMap
	ConfigMapKeyNginxConfig = "default.conf"
)

// Volume names
const (
	// VolumeNameNginxConfig is the name of nginx config volume
	VolumeNameNginxConfig = "nginx-config"
	// VolumeNameJWKSData is the name of JWKS data volume
	VolumeNameJWKSData = "jwks-data"
)

// Annotation keys for tracking ConfigMap changes
const (
	// AnnotationNginxConfigMapHash is the annotation key for nginx ConfigMap hash
	AnnotationNginxConfigMapHash = "jwks-operator.example.com/nginx-configmap-hash"
	// AnnotationJWKSConfigMapHash is the annotation key for JWKS ConfigMap hash
	AnnotationJWKSConfigMapHash = "jwks-operator.example.com/jwks-configmap-hash"
)

// Secret keys
const (
	// SecretKeyTLSKey is the key for TLS private key in Secret
	SecretKeyTLSKey = "tls.key"
	// SecretKeyTLSCert is the key for TLS certificate in Secret
	SecretKeyTLSCert = "tls.crt"
)
