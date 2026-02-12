package nginx

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// NginxResources is an alias for config.NginxResources for convenience
//
//nolint:revive // Naming stutter is acceptable for type aliases
type NginxResources = config.NginxResources

// Manager manages nginx ConfigMap, Deployment and Service resources
type Manager struct {
	client            client.Client
	generator         *ConfigGenerator
	deploymentManager *DeploymentManager
	serviceManager    *ServiceManager
	nginxConfig       *config.NginxConfig
}

// NewManager creates a new nginx manager
func NewManager(client client.Client, nginxConfig *config.NginxConfig) *Manager {
	cacheMaxAge := config.DefaultCacheMaxAge
	if nginxConfig != nil && nginxConfig.CacheMaxAge > 0 {
		cacheMaxAge = nginxConfig.CacheMaxAge
	}

	return &Manager{
		client:            client,
		generator:         NewConfigGenerator(cacheMaxAge),
		deploymentManager: NewDeploymentManager(client, nginxConfig),
		serviceManager:    NewServiceManager(client),
		nginxConfig:       nginxConfig,
	}
}

// UpdateConfig updates nginx ConfigMap with configuration
func (m *Manager) UpdateConfig(ctx context.Context, namespace, configMapName string, jwksConfigMapName string, endpoint string) error {
	if configMapName == "" {
		return fmt.Errorf("nginx ConfigMap name cannot be empty")
	}
	if jwksConfigMapName == "" {
		return fmt.Errorf("JWKS ConfigMap name cannot be empty")
	}

	// Generate nginx configuration
	nginxConfigContent, err := m.generator.GenerateConfig(jwksConfigMapName, endpoint)
	if err != nil {
		return fmt.Errorf("failed to generate nginx config: %w", err)
	}

	// Get or create ConfigMap
	nginxConfigMap := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: configMapName}

	err = m.client.Get(ctx, key, nginxConfigMap)
	if apierrors.IsNotFound(err) {
		// Create new ConfigMap
		nginxConfigMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				config.ConfigMapKeyNginxConfig: nginxConfigContent,
			},
		}
		return m.client.Create(ctx, nginxConfigMap)
	}
	if err != nil {
		return fmt.Errorf("failed to get nginx ConfigMap: %w", err)
	}

	// Update existing ConfigMap
	if nginxConfigMap.Data == nil {
		nginxConfigMap.Data = make(map[string]string)
	}

	// Always update ConfigMap to ensure it matches the current generator logic
	// This ensures that after operator restart/update, ConfigMap will be updated
	// even if the generated config looks similar but has different formatting or structure
	configKey := config.ConfigMapKeyNginxConfig
	oldConfig := nginxConfigMap.Data[configKey]
	if oldConfig != nginxConfigContent {
		nginxConfigMap.Data[configKey] = nginxConfigContent
		if err := m.client.Update(ctx, nginxConfigMap); err != nil {
			return fmt.Errorf("failed to update nginx ConfigMap: %w", err)
		}
		// ConfigMap updated, return to trigger Deployment update
		return nil
	}

	// ConfigMap content matches, no update needed
	return nil
}

// GetConfig retrieves nginx configuration from ConfigMap
func (m *Manager) GetConfig(ctx context.Context, namespace, configMapName string) (string, error) {
	configMap := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: namespace, Name: configMapName}

	err := m.client.Get(ctx, key, configMap)
	if apierrors.IsNotFound(err) {
		return "", nil // ConfigMap doesn't exist
	}
	if err != nil {
		return "", fmt.Errorf("failed to get nginx ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		return "", fmt.Errorf("nginx config not found in ConfigMap")
	}

	configKey := config.ConfigMapKeyNginxConfig
	nginxConfigContent, ok := configMap.Data[configKey]
	if !ok {
		return "", fmt.Errorf("%s not found in ConfigMap", configKey)
	}

	return nginxConfigContent, nil
}

// CreateConfigMap creates a new nginx ConfigMap
func (m *Manager) CreateConfigMap(ctx context.Context, namespace, configMapName string, nginxConfigContent string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			config.ConfigMapKeyNginxConfig: nginxConfigContent,
		},
	}

	return m.client.Create(ctx, configMap)
}

// EnsureDeployment ensures nginx Deployment exists for JWKS
func (m *Manager) EnsureDeployment(
	ctx context.Context,
	namespace, jwksName, nginxConfigMapName, jwksConfigMapName, endpoint string,
	nginxResources *NginxResources,
) error {
	return m.deploymentManager.EnsureDeployment(ctx, namespace, jwksName, nginxConfigMapName, jwksConfigMapName, endpoint, nginxResources)
}

// DeleteDeployment deletes nginx Deployment for JWKS
func (m *Manager) DeleteDeployment(ctx context.Context, namespace, jwksName string) error {
	return m.deploymentManager.DeleteDeployment(ctx, namespace, jwksName)
}

// EnsureService ensures nginx Service exists for JWKS
func (m *Manager) EnsureService(
	ctx context.Context,
	namespace, jwksName string,
) error {
	return m.serviceManager.EnsureService(ctx, namespace, jwksName)
}

// DeleteService deletes nginx Service for JWKS
func (m *Manager) DeleteService(ctx context.Context, namespace, jwksName string) error {
	return m.serviceManager.DeleteService(ctx, namespace, jwksName)
}
