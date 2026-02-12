package configmap

import (
	"context"
	"fmt"

	"github.com/jwks-operator/jwks-operator/pkg/jwks"
)

// UpdateStrategy defines how to update ConfigMap
type UpdateStrategy struct {
	manager *Manager
}

// NewUpdateStrategy creates a new update strategy
func NewUpdateStrategy(manager *Manager) *UpdateStrategy {
	return &UpdateStrategy{
		manager: manager,
	}
}

// Apply applies the update strategy
func (s *UpdateStrategy) Apply(ctx context.Context, namespace, configMapName string, newJWKS *jwks.JWKS, strategy string, keepOldKeys bool) error {
	if newJWKS == nil {
		return fmt.Errorf("new JWKS is nil")
	}

	switch strategy {
	case "rolling":
		return s.applyRollingStrategy(ctx, namespace, configMapName, newJWKS, keepOldKeys)
	case "immediate":
		return s.applyImmediateStrategy(ctx, namespace, configMapName, newJWKS)
	default:
		return fmt.Errorf("unknown update strategy: %s", strategy)
	}
}

// applyRollingStrategy applies rolling update strategy (graceful rotation)
func (s *UpdateStrategy) applyRollingStrategy(ctx context.Context, namespace, configMapName string, newJWKS *jwks.JWKS, keepOldKeys bool) error {
	if keepOldKeys {
		// Get current JWKS
		oldJWKS, err := s.manager.GetJWKS(ctx, namespace, configMapName)
		if err != nil {
			return fmt.Errorf("failed to get current JWKS: %w", err)
		}

		// Merge old and new keys
		if oldJWKS != nil {
			generator := jwks.NewGenerator()
			mergedJWKS, err := generator.MergeJWKS(oldJWKS, newJWKS)
			if err != nil {
				return fmt.Errorf("failed to merge JWKS: %w", err)
			}
			newJWKS = mergedJWKS
		}
	}

	// Update ConfigMap
	return s.manager.UpdateJWKS(ctx, namespace, configMapName, newJWKS)
}

// applyImmediateStrategy applies immediate update strategy (replace all keys)
func (s *UpdateStrategy) applyImmediateStrategy(ctx context.Context, namespace, configMapName string, newJWKS *jwks.JWKS) error {
	return s.manager.UpdateJWKS(ctx, namespace, configMapName, newJWKS)
}

// ShouldUpdate determines if an update is needed
func (s *UpdateStrategy) ShouldUpdate(oldJWKS, newJWKS *jwks.JWKS) bool {
	if oldJWKS == nil {
		return true
	}

	if newJWKS == nil || len(newJWKS.Keys) == 0 {
		return false
	}

	// Check if new key exists in old JWKS
	oldKids := make(map[string]bool)
	for _, key := range oldJWKS.Keys {
		oldKids[key.Kid] = true
	}

	// Update if any new key is not in old JWKS
	for _, key := range newJWKS.Keys {
		if !oldKids[key.Kid] {
			return true
		}
	}

	return false
}
