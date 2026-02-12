package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Load loads configuration from file and environment variables
// namespace must be provided from Kubernetes environment (e.g., from Pod namespace)
func Load(configPath string, namespace string) (*Config, error) {
	cfg := DefaultConfig()

	// Set namespace from Kubernetes environment (required, no defaults)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required and must be provided from Kubernetes environment")
	}
	cfg.Namespace = namespace

	// Load from file if exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	if err := LoadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %w", err)
	}

	// Validate configuration
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv(cfg *Config) error {
	prefix := "JWKS_OPERATOR_"

	// Helper function to parse duration
	parseDuration := func(key string) (time.Duration, error) {
		val := os.Getenv(key)
		if val == "" {
			return 0, nil
		}
		return time.ParseDuration(val)
	}

	// Helper function to get env var
	getEnv := func(key string) string {
		return os.Getenv(prefix + key)
	}

	// Load basic settings
	if val, err := parseDuration(prefix + "RECONCILE_INTERVAL"); err == nil && val > 0 {
		cfg.ReconcileInterval = Duration{Duration: val}
	}

	if val, err := parseDuration(prefix + "JWKS_UPDATE_INTERVAL"); err == nil && val > 0 {
		cfg.JWKSUpdateInterval = Duration{Duration: val}
	}

	// Load logging settings
	if val := getEnv("LOGGING_LEVEL"); val != "" {
		cfg.Logging.Level = val
	}

	if val := getEnv("LOGGING_FORMAT"); val != "" {
		cfg.Logging.Format = val
	}

	// Load metrics settings
	if val := getEnv("METRICS_PORT"); val != "" {
		var port int
		if _, err := fmt.Sscanf(val, "%d", &port); err == nil {
			cfg.Metrics.Port = port
		}
	}

	if val := getEnv("METRICS_PATH"); val != "" {
		cfg.Metrics.Path = val
	}

	return nil
}

// Validate validates the configuration
func Validate(cfg *Config) error {
	// Namespace is required (must be set from Kubernetes environment)
	if cfg.Namespace == "" {
		return fmt.Errorf("namespace is required and must be provided from Kubernetes environment")
	}

	// ReconcileInterval is required
	if cfg.ReconcileInterval.Duration <= 0 {
		return fmt.Errorf("reconcileInterval is required and must be positive")
	}

	// JWKSUpdateInterval is required
	if cfg.JWKSUpdateInterval.Duration <= 0 {
		return fmt.Errorf("jwksUpdateInterval is required and must be positive")
	}

	// MaxOldKeys must be non-negative
	if cfg.MaxOldKeys < 0 {
		return fmt.Errorf("maxOldKeys must be non-negative")
	}

	// DefaultOldKeysTTL is required
	if cfg.DefaultOldKeysTTL.Duration <= 0 {
		return fmt.Errorf("defaultOldKeysTTL is required and must be positive")
	}

	// DefaultUpdateStrategy is required
	if cfg.DefaultUpdateStrategy != "rolling" && cfg.DefaultUpdateStrategy != "immediate" {
		return fmt.Errorf("defaultUpdateStrategy is required and must be 'rolling' or 'immediate'")
	}

	// Validate logging level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(cfg.Logging.Level)] {
		return fmt.Errorf("logging level must be one of: debug, info, warn, error")
	}

	// Validate nginx configuration
	if err := validateNginxConfig(&cfg.Nginx); err != nil {
		return fmt.Errorf("nginx configuration validation failed: %w", err)
	}

	// Validate verification configuration
	if err := validateVerificationConfig(&cfg.Verification); err != nil {
		return fmt.Errorf("verification configuration validation failed: %w", err)
	}

	return nil
}

// validateNginxConfig validates nginx configuration
func validateNginxConfig(nginxConfig *NginxConfig) error {
	if nginxConfig == nil {
		return nil // Nil config is valid (will use defaults)
	}

	// Validate port
	if nginxConfig.Port > 0 && (nginxConfig.Port < 1 || nginxConfig.Port > 65535) {
		return fmt.Errorf("nginx port must be between 1 and 65535, got %d", nginxConfig.Port)
	}

	// Validate replicas
	if nginxConfig.Replicas < 0 {
		return fmt.Errorf("nginx replicas must be non-negative, got %d", nginxConfig.Replicas)
	}

	// Validate cache max age
	if nginxConfig.CacheMaxAge < 0 {
		return fmt.Errorf("nginx cache max age must be non-negative, got %d", nginxConfig.CacheMaxAge)
	}

	return nil
}

// validateVerificationConfig validates verification configuration
func validateVerificationConfig(verificationConfig *VerificationConfig) error {
	if verificationConfig == nil {
		return nil // Nil config is valid (will use defaults)
	}

	// Validate retry count
	if verificationConfig.RetryCount < 0 {
		return fmt.Errorf("verification retry count must be non-negative, got %d", verificationConfig.RetryCount)
	}

	return nil
}
