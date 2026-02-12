package config

import "time"

// Config represents the operator configuration
type Config struct {
	// Namespace is the namespace where the operator is running
	// This is determined from the Kubernetes environment, not from config file
	Namespace string `yaml:"-"`

	// ReconcileInterval is the interval between reconciliations
	ReconcileInterval Duration `yaml:"reconcileInterval"`

	// JWKSUpdateInterval is the interval for checking JWKS updates
	JWKSUpdateInterval Duration `yaml:"jwksUpdateInterval"`

	// JWKSVerificationInterval is the interval for verifying JWKS from nginx
	JWKSVerificationInterval Duration `yaml:"jwksVerificationInterval"`

	// MaxOldKeys is the maximum number of old keys in JWKS
	MaxOldKeys int `yaml:"maxOldKeys"`

	// DefaultOldKeysTTL is the default TTL for old keys
	DefaultOldKeysTTL Duration `yaml:"defaultOldKeysTTL"`

	// DefaultUpdateStrategy is the default update strategy
	DefaultUpdateStrategy string `yaml:"defaultUpdateStrategy"`

	// DefaultKeepOldKeys determines if old keys should be kept by default
	DefaultKeepOldKeys bool `yaml:"defaultKeepOldKeys"`

	// CleanupOnDelete determines if ConfigMap should be deleted when JWKS is deleted
	CleanupOnDelete bool `yaml:"cleanupOnDelete"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics"`

	// Health check configuration
	Health HealthConfig `yaml:"health"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry"`

	// Cache configuration
	Cache CacheConfig `yaml:"cache"`

	// Validation configuration
	Validation ValidationConfig `yaml:"validation"`

	// Security configuration
	Security SecurityConfig `yaml:"security"`

	// Nginx configuration
	Nginx NginxConfig `yaml:"nginx"`

	// Verification configuration
	Verification VerificationConfig `yaml:"verification"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level            string `yaml:"level"`
	Format           string `yaml:"format"`
	VerboseReconcile bool   `yaml:"verboseReconcile"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Port     int    `yaml:"port"`
	Path     string `yaml:"path"`
	Detailed bool   `yaml:"detailed"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Port          int    `yaml:"port"`
	LivenessPath  string `yaml:"livenessPath"`
	ReadinessPath string `yaml:"readinessPath"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts       int      `yaml:"maxAttempts"`
	InitialDelay      Duration `yaml:"initialDelay"`
	MaxDelay          Duration `yaml:"maxDelay"`
	BackoffMultiplier float64  `yaml:"backoffMultiplier"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	EnableJWKSCache bool     `yaml:"enableJWKSCache"`
	JWKSCacheTTL    Duration `yaml:"jwksCacheTTL"`
	EnableCertCache bool     `yaml:"enableCertCache"`
	CertCacheTTL    Duration `yaml:"certCacheTTL"`
}

// ValidationConfig represents validation configuration
type ValidationConfig struct {
	ValidateCertificates bool `yaml:"validateCertificates"`
	ValidateJWKS         bool `yaml:"validateJWKS"`
	ValidateJSON         bool `yaml:"validateJSON"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	MinimalRBAC          bool `yaml:"minimalRBAC"`
	CheckOwnerReferences bool `yaml:"checkOwnerReferences"`
	UseServiceAccount    bool `yaml:"useServiceAccount"`
}

// NginxConfig represents nginx server configuration
type NginxConfig struct {
	// Image is the nginx container image
	Image string `yaml:"image"`
	// Port is the nginx port
	Port int `yaml:"port"`
	// Replicas is the number of nginx replicas
	Replicas int32 `yaml:"replicas"`
	// Resources represents nginx container resources
	Resources NginxResources `yaml:"resources"`
	// CacheMaxAge is the cache max-age in seconds for nginx responses
	CacheMaxAge int `yaml:"cacheMaxAge"`
}

// VerificationConfig represents JWKS verification configuration
type VerificationConfig struct {
	// Timeout is the timeout for HTTP requests during verification
	Timeout time.Duration `yaml:"timeout"`
	// RetryCount is the number of retry attempts
	RetryCount int `yaml:"retryCount"`
	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `yaml:"retryDelay"`
	// ContextTimeout is the context timeout for verification
	ContextTimeout time.Duration `yaml:"contextTimeout"`
}

// NginxResources represents nginx container resources
type NginxResources struct {
	Requests NginxResourceRequirements `yaml:"requests"`
	Limits   NginxResourceRequirements `yaml:"limits"`
}

// NginxResourceRequirements represents CPU and memory requirements
type NginxResourceRequirements struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}
