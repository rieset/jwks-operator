package nginx

import (
	"fmt"
	"strings"
)

const (
	// DefaultEndpoint is the default JWKS endpoint path (kept for backward compatibility)
	// JWKS is always available at both "/" and "/jwks.json" paths
	DefaultEndpoint = "/jwks.json"
)

// ValidateEndpoint validates an endpoint path
func ValidateEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	if !strings.HasPrefix(endpoint, "/") {
		return fmt.Errorf("endpoint must start with '/'")
	}

	return nil
}

// NormalizeEndpoint normalizes an endpoint path
func NormalizeEndpoint(endpoint string) string {
	if endpoint == "" {
		return DefaultEndpoint
	}

	// Ensure it starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	return endpoint
}
