package nginx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// DeploymentManager manages nginx Deployment resources
type DeploymentManager struct {
	client client.Client
	config *config.NginxConfig
}

// NewDeploymentManager creates a new nginx Deployment manager
func NewDeploymentManager(client client.Client, nginxConfig *config.NginxConfig) *DeploymentManager {
	return &DeploymentManager{
		client: client,
		config: nginxConfig,
	}
}

// EnsureDeployment ensures that nginx Deployment exists for JWKS
func (m *DeploymentManager) EnsureDeployment(
	ctx context.Context,
	namespace string,
	jwksConfigName string,
	nginxConfigMapName string,
	jwksConfigMapName string,
	endpoint string,
	nginxResources *NginxResources,
) error {
	deploymentName := jwksConfigName

	// Check if Deployment already exists
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Namespace: namespace, Name: deploymentName}

	err := m.client.Get(ctx, key, deployment)
	if err == nil {
		// Deployment exists, check if it needs update
		return m.updateDeploymentIfNeeded(ctx, deployment, nginxConfigMapName, jwksConfigMapName, nginxResources)
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get ConfigMaps to compute their hashes for initial annotations
	nginxConfigMap := &corev1.ConfigMap{}
	nginxKey := types.NamespacedName{Namespace: namespace, Name: nginxConfigMapName}
	if err := m.client.Get(ctx, nginxKey, nginxConfigMap); err != nil {
		return fmt.Errorf("failed to get nginx ConfigMap: %w", err)
	}

	jwksConfigMap := &corev1.ConfigMap{}
	jwksKey := types.NamespacedName{Namespace: namespace, Name: jwksConfigMapName}
	if err := m.client.Get(ctx, jwksKey, jwksConfigMap); err != nil {
		return fmt.Errorf("failed to get JWKS ConfigMap: %w", err)
	}

	// Create new Deployment
	deployment = m.createDeployment(deploymentName, namespace, nginxConfigMapName, jwksConfigMapName, endpoint, nginxResources)

	// Set initial ConfigMap hashes in annotations
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations[config.AnnotationNginxConfigMapHash] = computeConfigMapHash(nginxConfigMap)
	deployment.Spec.Template.Annotations[config.AnnotationJWKSConfigMapHash] = computeConfigMapHash(jwksConfigMap)

	return m.client.Create(ctx, deployment)
}

// computeConfigMapHash computes SHA256 hash of ConfigMap data
func computeConfigMapHash(configMap *corev1.ConfigMap) string {
	if configMap == nil {
		return ""
	}

	// Combine all data and binary data into a single string for hashing
	var dataStr string
	if configMap.Data != nil {
		for k, v := range configMap.Data {
			dataStr += k + "=" + v + "\n"
		}
	}
	if configMap.BinaryData != nil {
		for k, v := range configMap.BinaryData {
			dataStr += k + "=" + string(v) + "\n"
		}
	}

	hash := sha256.Sum256([]byte(dataStr))
	return hex.EncodeToString(hash[:])
}

// updateDeploymentIfNeeded updates Deployment if ConfigMaps changed
func (m *DeploymentManager) updateDeploymentIfNeeded(
	ctx context.Context,
	deployment *appsv1.Deployment,
	nginxConfigMapName, jwksConfigMapName string,
	nginxResources *NginxResources,
) error {
	needsUpdate := false

	// Get current ConfigMaps to compute their hashes
	nginxConfigMap := &corev1.ConfigMap{}
	nginxKey := types.NamespacedName{Namespace: deployment.Namespace, Name: nginxConfigMapName}
	if err := m.client.Get(ctx, nginxKey, nginxConfigMap); err != nil {
		return fmt.Errorf("failed to get nginx ConfigMap: %w", err)
	}

	jwksConfigMap := &corev1.ConfigMap{}
	jwksKey := types.NamespacedName{Namespace: deployment.Namespace, Name: jwksConfigMapName}
	if err := m.client.Get(ctx, jwksKey, jwksConfigMap); err != nil {
		return fmt.Errorf("failed to get JWKS ConfigMap: %w", err)
	}

	// Compute current hashes
	nginxHash := computeConfigMapHash(nginxConfigMap)
	jwksHash := computeConfigMapHash(jwksConfigMap)

	// Initialize annotations if needed
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}

	// Check if ConfigMap hashes changed
	annotationNginxHash := config.AnnotationNginxConfigMapHash
	annotationJWKSHash := config.AnnotationJWKSConfigMapHash

	if deployment.Spec.Template.Annotations[annotationNginxHash] != nginxHash {
		deployment.Spec.Template.Annotations[annotationNginxHash] = nginxHash
		needsUpdate = true
	}

	if deployment.Spec.Template.Annotations[annotationJWKSHash] != jwksHash {
		deployment.Spec.Template.Annotations[annotationJWKSHash] = jwksHash
		needsUpdate = true
	}

	// Check nginx config volume name
	if len(deployment.Spec.Template.Spec.Volumes) > 0 {
		for i, vol := range deployment.Spec.Template.Spec.Volumes {
			if vol.Name == config.VolumeNameNginxConfig && vol.ConfigMap != nil {
				if vol.ConfigMap.Name != nginxConfigMapName {
					deployment.Spec.Template.Spec.Volumes[i].ConfigMap.Name = nginxConfigMapName
					needsUpdate = true
				}
			}
			if vol.Name == config.VolumeNameJWKSData && vol.ConfigMap != nil {
				if vol.ConfigMap.Name != jwksConfigMapName {
					deployment.Spec.Template.Spec.Volumes[i].ConfigMap.Name = jwksConfigMapName
					needsUpdate = true
				}
			}
		}
	}

	// Check if resources need update
	if nginxResources != nil {
		container := &deployment.Spec.Template.Spec.Containers[0]
		newResources := m.buildResourceRequirements(nginxResources)
		if !resourcesEqual(container.Resources, newResources) {
			container.Resources = newResources
			needsUpdate = true
		}
	}

	if needsUpdate {
		// Trigger rolling update by updating annotation
		deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")
		return m.client.Update(ctx, deployment)
	}

	return nil
}

// DeleteDeployment deletes nginx Deployment
func (m *DeploymentManager) DeleteDeployment(ctx context.Context, namespace, jwksConfigName string) error {
	deploymentName := jwksConfigName
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
	}

	err := m.client.Delete(ctx, deployment)
	if apierrors.IsNotFound(err) {
		return nil // Already deleted, nothing to do
	}
	return err
}

// GetDeploymentName returns the name of nginx Deployment for JWKS
func GetDeploymentName(jwksConfigName string) string {
	return jwksConfigName
}
