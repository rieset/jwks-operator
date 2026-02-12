package configmap

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/pkg/jwks"
)

// Manager manages ConfigMap resources with JWKS data
type Manager struct {
	client client.Client
}

// NewManager creates a new ConfigMap manager
func NewManager(client client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// UpdateJWKS updates a ConfigMap with JWKS data
func (m *Manager) UpdateJWKS(ctx context.Context, namespace, configMapName string, jwksData *jwks.JWKS) error {
	if jwksData == nil {
		return fmt.Errorf("JWKS data is nil")
	}

	// Convert JWKS to JSON
	jsonData, err := jwks.ToJSON(jwksData)
	if err != nil {
		return fmt.Errorf("failed to convert JWKS to JSON: %w", err)
	}

	// Get or create ConfigMap
	configMap := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: configMapName}

	err = m.client.Get(ctx, key, configMap)
	if apierrors.IsNotFound(err) {
		// Create new ConfigMap
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			BinaryData: map[string][]byte{
				"jwks.json": jsonData,
			},
		}
		return m.client.Create(ctx, configMap)
	}
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update existing ConfigMap
	if configMap.BinaryData == nil {
		configMap.BinaryData = make(map[string][]byte)
	}
	configMap.BinaryData["jwks.json"] = jsonData

	return m.client.Update(ctx, configMap)
}

// GetJWKS retrieves JWKS from a ConfigMap
func (m *Manager) GetJWKS(ctx context.Context, namespace, configMapName string) (*jwks.JWKS, error) {
	configMap := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: configMapName}

	err := m.client.Get(ctx, key, configMap)
	if apierrors.IsNotFound(err) {
		return nil, nil // ConfigMap doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Extract JWKS JSON
	var jsonData []byte
	if configMap.BinaryData != nil {
		jsonData = configMap.BinaryData["jwks.json"]
	} else if configMap.Data != nil {
		jsonData = []byte(configMap.Data["jwks.json"])
	}

	if len(jsonData) == 0 {
		return nil, fmt.Errorf("jwks.json not found in ConfigMap")
	}

	// Parse JSON
	var jwksData jwks.JWKS
	if err := json.Unmarshal(jsonData, &jwksData); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS JSON: %w", err)
	}

	return &jwksData, nil
}

// CreateConfigMap creates a new ConfigMap with JWKS data
func (m *Manager) CreateConfigMap(ctx context.Context, namespace, configMapName string, jwksData *jwks.JWKS) error {
	return m.UpdateJWKS(ctx, namespace, configMapName, jwksData)
}
