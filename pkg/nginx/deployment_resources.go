package nginx

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// buildResourceRequirements builds ResourceRequirements from NginxResources config
func (m *DeploymentManager) buildResourceRequirements(nginxResources *NginxResources) corev1.ResourceRequirements {
	// Use defaults from config or constants
	cpuRequest := config.DefaultNginxCPURequest
	memoryRequest := config.DefaultNginxMemoryRequest
	cpuLimit := config.DefaultNginxCPULimit
	memoryLimit := config.DefaultNginxMemoryLimit

	// Override with config if available
	if m.config != nil && m.config.Resources.Requests.CPU != "" {
		cpuRequest = m.config.Resources.Requests.CPU
	}
	if m.config != nil && m.config.Resources.Requests.Memory != "" {
		memoryRequest = m.config.Resources.Requests.Memory
	}
	if m.config != nil && m.config.Resources.Limits.CPU != "" {
		cpuLimit = m.config.Resources.Limits.CPU
	}
	if m.config != nil && m.config.Resources.Limits.Memory != "" {
		memoryLimit = m.config.Resources.Limits.Memory
	}

	// Override with provided resources if available
	if nginxResources != nil {
		if nginxResources.Requests.CPU != "" {
			cpuRequest = nginxResources.Requests.CPU
		}
		if nginxResources.Requests.Memory != "" {
			memoryRequest = nginxResources.Requests.Memory
		}
		if nginxResources.Limits.CPU != "" {
			cpuLimit = nginxResources.Limits.CPU
		}
		if nginxResources.Limits.Memory != "" {
			memoryLimit = nginxResources.Limits.Memory
		}
	}

	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpuRequest),
			corev1.ResourceMemory: resource.MustParse(memoryRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpuLimit),
			corev1.ResourceMemory: resource.MustParse(memoryLimit),
		},
	}
}

// resourcesEqual checks if two ResourceRequirements are equal
func resourcesEqual(a, b corev1.ResourceRequirements) bool {
	return a.Requests.Cpu().Equal(*b.Requests.Cpu()) &&
		a.Requests.Memory().Equal(*b.Requests.Memory()) &&
		a.Limits.Cpu().Equal(*b.Limits.Cpu()) &&
		a.Limits.Memory().Equal(*b.Limits.Memory())
}
