package utils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureConfigMapExists checks if ConfigMap exists, returns true if exists, false if not found
// Returns error if there was an error checking (other than NotFound)
func EnsureConfigMapExists(ctx context.Context, client client.Client, namespace, name string) (bool, error) {
	configMap := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: name}

	err := client.Get(ctx, key, configMap)
	if err == nil {
		return true, nil
	}

	if apierrors.IsNotFound(err) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check ConfigMap: %w", err)
}

// GetConfigMap retrieves ConfigMap, returns nil if not found
func GetConfigMap(ctx context.Context, client client.Client, namespace, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: name}

	err := client.Get(ctx, key, configMap)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	return configMap, nil
}
