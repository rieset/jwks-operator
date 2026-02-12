package validation

import (
	"fmt"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// Validator validates configuration and resources
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateNginxConfig validates nginx configuration
func (v *Validator) ValidateNginxConfig(nginxConfig *config.NginxConfig) error {
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

	// Validate resources if provided
	if err := v.ValidateNginxResources(&nginxConfig.Resources); err != nil {
		return fmt.Errorf("invalid nginx resources: %w", err)
	}

	return nil
}

// ValidateNginxResources validates nginx resource requirements
func (v *Validator) ValidateNginxResources(resources *config.NginxResources) error {
	if resources == nil {
		return nil // Nil resources is valid (will use defaults)
	}

	// Validate CPU requests
	if resources.Requests.CPU != "" {
		if err := v.validateCPU(resources.Requests.CPU); err != nil {
			return fmt.Errorf("invalid CPU request: %w", err)
		}
	}

	// Validate memory requests
	if resources.Requests.Memory != "" {
		if err := v.validateMemory(resources.Requests.Memory); err != nil {
			return fmt.Errorf("invalid memory request: %w", err)
		}
	}

	// Validate CPU limits
	if resources.Limits.CPU != "" {
		if err := v.validateCPU(resources.Limits.CPU); err != nil {
			return fmt.Errorf("invalid CPU limit: %w", err)
		}
	}

	// Validate memory limits
	if resources.Limits.Memory != "" {
		if err := v.validateMemory(resources.Limits.Memory); err != nil {
			return fmt.Errorf("invalid memory limit: %w", err)
		}
	}

	return nil
}

// validateCPU validates CPU resource string (e.g., "100m", "1")
func (v *Validator) validateCPU(cpu string) error {
	if cpu == "" {
		return fmt.Errorf("CPU value cannot be empty")
	}

	// Basic validation - Kubernetes will do more detailed validation
	// We just check that it's not obviously wrong
	if len(cpu) == 0 {
		return fmt.Errorf("CPU value cannot be empty")
	}

	return nil
}

// validateMemory validates memory resource string (e.g., "128Mi", "1Gi")
func (v *Validator) validateMemory(memory string) error {
	if memory == "" {
		return fmt.Errorf("memory value cannot be empty")
	}

	// Basic validation - Kubernetes will do more detailed validation
	// We just check that it's not obviously wrong
	if len(memory) == 0 {
		return fmt.Errorf("memory value cannot be empty")
	}

	return nil
}

// ValidateVerificationConfig validates verification configuration
func (v *Validator) ValidateVerificationConfig(verificationConfig *config.VerificationConfig) error {
	if verificationConfig == nil {
		return nil // Nil config is valid (will use defaults)
	}

	// Validate retry count
	if verificationConfig.RetryCount < 0 {
		return fmt.Errorf("verification retry count must be non-negative, got %d", verificationConfig.RetryCount)
	}

	// Validate timeout (must be positive)
	if verificationConfig.Timeout < 0 {
		return fmt.Errorf("verification timeout must be non-negative, got %v", verificationConfig.Timeout)
	}

	return nil
}
