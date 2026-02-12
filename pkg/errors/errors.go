package errors

import (
	"fmt"
)

// Error represents a typed error in the JWKS operator
type Error struct {
	Type    ErrorType
	Message string
	Err     error
}

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeSecretNotFound indicates that a required Secret was not found
	ErrorTypeSecretNotFound ErrorType = "SecretNotFound"
	// ErrorTypeJWKSGenerationFailed indicates that JWKS generation failed
	ErrorTypeJWKSGenerationFailed ErrorType = "JWKSGenerationFailed"
	// ErrorTypeConfigMapUpdateFailed indicates that ConfigMap update failed
	ErrorTypeConfigMapUpdateFailed ErrorType = "ConfigMapUpdateFailed"
	// ErrorTypeNginxConfigUpdateFailed indicates that nginx config update failed
	ErrorTypeNginxConfigUpdateFailed ErrorType = "NginxConfigUpdateFailed"
	// ErrorTypeNginxDeploymentFailed indicates that nginx deployment failed
	ErrorTypeNginxDeploymentFailed ErrorType = "NginxDeploymentFailed"
	// ErrorTypeNginxServiceFailed indicates that nginx service failed
	ErrorTypeNginxServiceFailed ErrorType = "NginxServiceFailed"
	// ErrorTypeJWKSVerificationFailed indicates that JWKS verification failed
	ErrorTypeJWKSVerificationFailed ErrorType = "JWKSVerificationFailed"
	// ErrorTypeInvalidConfiguration indicates that configuration is invalid
	ErrorTypeInvalidConfiguration ErrorType = "InvalidConfiguration"
	// ErrorTypeResourceNotFound indicates that a required resource was not found
	ErrorTypeResourceNotFound ErrorType = "ResourceNotFound"
)

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new typed error
func NewError(errType ErrorType, message string, err error) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our typed error
	if opErr, ok := err.(*Error); ok {
		switch opErr.Type {
		case ErrorTypeSecretNotFound, ErrorTypeResourceNotFound:
			// Resource not found might be temporary (e.g., resource being created)
			return true
		case ErrorTypeJWKSVerificationFailed:
			// Verification failures are usually temporary
			return true
		case ErrorTypeNginxDeploymentFailed, ErrorTypeNginxServiceFailed:
			// Deployment/service failures might be temporary
			return true
		default:
			return false
		}
	}

	// Default: assume non-retryable
	return false
}

// Helper functions for common error types

// NewSecretNotFoundError creates a new SecretNotFound error
func NewSecretNotFoundError(secretName string, err error) *Error {
	return NewError(ErrorTypeSecretNotFound, fmt.Sprintf("Secret %s not found", secretName), err)
}

// NewJWKSGenerationError creates a new JWKSGenerationFailed error
func NewJWKSGenerationError(err error) *Error {
	return NewError(ErrorTypeJWKSGenerationFailed, "Failed to generate JWKS", err)
}

// NewConfigMapUpdateError creates a new ConfigMapUpdateFailed error
func NewConfigMapUpdateError(configMapName string, err error) *Error {
	return NewError(ErrorTypeConfigMapUpdateFailed, fmt.Sprintf("Failed to update ConfigMap %s", configMapName), err)
}

// NewNginxConfigUpdateError creates a new NginxConfigUpdateFailed error
func NewNginxConfigUpdateError(err error) *Error {
	return NewError(ErrorTypeNginxConfigUpdateFailed, "Failed to update nginx config", err)
}

// NewNginxDeploymentError creates a new NginxDeploymentFailed error
func NewNginxDeploymentError(err error) *Error {
	return NewError(ErrorTypeNginxDeploymentFailed, "Failed to ensure nginx deployment", err)
}

// NewNginxServiceError creates a new NginxServiceFailed error
func NewNginxServiceError(err error) *Error {
	return NewError(ErrorTypeNginxServiceFailed, "Failed to ensure nginx service", err)
}

// NewJWKSVerificationError creates a new JWKSVerificationFailed error
func NewJWKSVerificationError(err error) *Error {
	return NewError(ErrorTypeJWKSVerificationFailed, "Failed to verify JWKS", err)
}

// NewInvalidConfigurationError creates a new InvalidConfiguration error
func NewInvalidConfigurationError(message string, err error) *Error {
	return NewError(ErrorTypeInvalidConfiguration, message, err)
}
