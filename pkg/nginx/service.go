package nginx

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// ServiceManager manages nginx Service resources
type ServiceManager struct {
	client client.Client
}

// NewServiceManager creates a new nginx Service manager
func NewServiceManager(client client.Client) *ServiceManager {
	return &ServiceManager{
		client: client,
	}
}

// EnsureService ensures that nginx Service exists for JWKS
func (m *ServiceManager) EnsureService(
	ctx context.Context,
	namespace string,
	jwksConfigName string,
) error {
	serviceName := jwksConfigName

	// Check if Service already exists
	service := &corev1.Service{}
	key := types.NamespacedName{Namespace: namespace, Name: serviceName}

	err := m.client.Get(ctx, key, service)
	if err == nil {
		// Service exists, check if selector matches Deployment PodTemplate labels
		// Get Deployment to check actual PodTemplate labels
		deploymentName := jwksConfigName
		deployment := &appsv1.Deployment{}
		deploymentKey := types.NamespacedName{Namespace: namespace, Name: deploymentName}

		// Try to get Deployment with exact name first
		deploymentErr := m.client.Get(ctx, deploymentKey, deployment)
		if deploymentErr != nil {
			// Try with nginx- prefix (for backward compatibility)
			deploymentKey.Name = "nginx-" + jwksConfigName
			deploymentErr = m.client.Get(ctx, deploymentKey, deployment)
		}

		if deploymentErr == nil {
			// Get actual labels from Deployment PodTemplate (not Deployment labels)
			podLabels := deployment.Spec.Template.Labels
			actualApp := podLabels[config.LabelApp]

			// Build expected selector - simplified to use only app label
			expectedSelector := map[string]string{
				config.LabelApp: jwksConfigName, // Use JWKS resource name as app label value
			}

			// If Deployment PodTemplate has app label, use it (for compatibility)
			if actualApp != "" {
				expectedSelector[config.LabelApp] = actualApp
			}

			// Update Service selector if it doesn't match
			if !selectorsEqual(service.Spec.Selector, expectedSelector) {
				service.Spec.Selector = expectedSelector
				return m.client.Update(ctx, service)
			}
		} else {
			// Deployment not found - update selector to simplified version
			expectedSelector := map[string]string{
				config.LabelApp: jwksConfigName,
			}
			if !selectorsEqual(service.Spec.Selector, expectedSelector) {
				service.Spec.Selector = expectedSelector
				return m.client.Update(ctx, service)
			}
		}

		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Create new Service with simplified selector (only app label)
	service = m.createService(serviceName, namespace, jwksConfigName)
	return m.client.Create(ctx, service)
}

// selectorsEqual checks if two selectors are equal
func selectorsEqual(s1, s2 map[string]string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for k, v := range s1 {
		if s2[k] != v {
			return false
		}
	}
	return true
}

// createService creates a new nginx Service spec
func (m *ServiceManager) createService(
	name, namespace, jwksConfigName string,
) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				config.LabelApp:        jwksConfigName, // Use JWKS resource name as app label value
				config.LabelJWKSConfig: jwksConfigName,
				config.LabelManagedBy:  config.LabelManagedByValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				config.LabelApp: jwksConfigName, // Simplified selector - only app label
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// DeleteService deletes nginx Service
func (m *ServiceManager) DeleteService(ctx context.Context, namespace, jwksConfigName string) error {
	serviceName := jwksConfigName
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
	}

	err := m.client.Delete(ctx, service)
	if apierrors.IsNotFound(err) {
		return nil // Already deleted, nothing to do
	}
	return err
}
