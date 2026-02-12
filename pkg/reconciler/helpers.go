package reconciler

import (
	"time"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/nginx"
)

// getUpdateStrategy returns the update strategy from CRD or config default
func (l *ReconciliationLoop) getUpdateStrategy(jwks *v1alpha1.JWKS) string {
	if jwks.Spec.UpdateStrategy != "" {
		return jwks.Spec.UpdateStrategy
	}
	return l.config.DefaultUpdateStrategy
}

// shouldKeepOldKeys determines if old keys should be kept
func (l *ReconciliationLoop) shouldKeepOldKeys(jwks *v1alpha1.JWKS) bool {
	// Use spec value if set, otherwise use config default
	if jwks.Spec.KeepOldKeys {
		return true
	}
	return l.config.DefaultKeepOldKeys
}

// getEndpoint returns the endpoint from CRD or default
func (l *ReconciliationLoop) getEndpoint(jwks *v1alpha1.JWKS) string {
	if jwks.Spec.Endpoint != "" {
		return jwks.Spec.Endpoint
	}
	return nginx.DefaultEndpoint
}

// getJWKSUpdateInterval returns JWKS update interval from CRD or config default
func (l *ReconciliationLoop) getJWKSUpdateInterval(jwks *v1alpha1.JWKS) time.Duration {
	if jwks.Spec.JWKSUpdateInterval != "" {
		if d, err := time.ParseDuration(jwks.Spec.JWKSUpdateInterval); err == nil && d > 0 {
			return d
		}
	}
	return l.config.JWKSUpdateInterval.Duration
}

// getJWKSVerificationInterval returns JWKS verification interval from CRD or config default
func (l *ReconciliationLoop) getJWKSVerificationInterval(jwks *v1alpha1.JWKS) time.Duration {
	if jwks.Spec.JWKSVerificationInterval != "" {
		if d, err := time.ParseDuration(jwks.Spec.JWKSVerificationInterval); err == nil && d > 0 {
			return d
		}
	}
	return l.config.JWKSVerificationInterval.Duration
}

// shouldVerifyJWKS determines if JWKS verification should be performed
// forceVerification can be used to force verification regardless of time elapsed (e.g., after operator restart)
func (l *ReconciliationLoop) shouldVerifyJWKS(jwks *v1alpha1.JWKS, forceVerification bool) bool {
	// Always verify if JWKSVerified is nil (first time)
	if jwks.Status.JWKSVerified == nil {
		return true
	}

	// Force verification if requested (e.g., after operator restart)
	if forceVerification {
		return true
	}

	// Check if enough time has passed since last verification
	elapsed := time.Since(jwks.Status.JWKSVerified.Time)
	verificationInterval := l.getJWKSVerificationInterval(jwks)
	return elapsed >= verificationInterval
}
