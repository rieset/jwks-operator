package configmap

import (
	"fmt"
	"time"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/jwks"
)

// KeyRotationManager manages key rotation in JWKS
type KeyRotationManager struct{}

// NewKeyRotationManager creates a new key rotation manager
func NewKeyRotationManager() *KeyRotationManager {
	return &KeyRotationManager{}
}

// AddNewKey adds a new key to JWKS
func (r *KeyRotationManager) AddNewKey(jwksData *jwks.JWKS, newKey *jwks.JWK) error {
	if jwksData == nil {
		return fmt.Errorf("JWKS is nil")
	}
	if newKey == nil {
		return fmt.Errorf("new key is nil")
	}

	// Check if key with same kid already exists
	for _, key := range jwksData.Keys {
		if key.Kid == newKey.Kid {
			return nil // Key already exists
		}
	}

	// Add new key
	jwksData.Keys = append(jwksData.Keys, *newKey)
	return nil
}

// RemoveExpiredKeys removes expired keys from JWKS based on TTL
// Note: This is a simplified implementation. In production, you might want to
// track key creation time in metadata or annotations
func (r *KeyRotationManager) RemoveExpiredKeys(jwksData *jwks.JWKS, ttl time.Duration) error {
	if jwksData == nil {
		return fmt.Errorf("JWKS is nil")
	}

	// For now, we keep all keys
	// In a full implementation, you would track key creation time
	// and remove keys older than TTL
	_ = ttl

	return nil
}

// ShouldKeepOldKeys determines if old keys should be kept based on JWKS spec
func (r *KeyRotationManager) ShouldKeepOldKeys(jwks *v1alpha1.JWKS) bool {
	if jwks == nil {
		return false
	}
	return jwks.Spec.KeepOldKeys
}
