package nginx

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// safeIntToInt32 safely converts int to int32, ensuring no overflow
// Port values are always in safe range (0-65535), so this is safe
//
//nolint:gosec // G115: We validate overflow before conversion
func safeIntToInt32(val int) int32 {
	const maxInt32 = 2147483647
	if val > maxInt32 {
		panic(fmt.Sprintf("port value %d exceeds int32 maximum", val))
	}
	return int32(val)
}

// buildContainerSpec builds the nginx container specification
func (m *DeploymentManager) buildContainerSpec(
	_ string, // name - not used in container spec
	image string,
	port int,
	_ string, // nginxConfigMapName - not used in container spec
	_ string, // jwksConfigMapName - not used in container spec
	nginxResources *NginxResources,
) corev1.Container {
	return corev1.Container{
		Name:    "nginx",
		Image:   image,
		Command: []string{"nginx", "-g", "daemon off;"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: safeIntToInt32(port),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: m.buildVolumeMounts(),
		Resources:    m.buildResourceRequirements(nginxResources),
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: config.HealthCheckPath,
					Port: intstr.FromInt(port),
				},
			},
			InitialDelaySeconds: config.DefaultLivenessProbeInitialDelay,
			PeriodSeconds:       config.DefaultLivenessProbePeriod,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: config.JWKSEndpointPath,
					Port: intstr.FromInt(port),
				},
			},
			InitialDelaySeconds: config.DefaultReadinessProbeInitialDelay,
			PeriodSeconds:       config.DefaultReadinessProbePeriod,
		},
	}
}

// buildVolumeMounts builds volume mounts for nginx container
func (m *DeploymentManager) buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      config.VolumeNameNginxConfig,
			MountPath: "/etc/nginx/conf.d",
			ReadOnly:  true,
		},
		{
			Name:      config.VolumeNameJWKSData,
			MountPath: "/usr/share/nginx/html/jwks.json",
			SubPath:   config.ConfigMapKeyJWKS,
			ReadOnly:  true,
		},
	}
}

// buildVolumes builds volumes for nginx pod
func (m *DeploymentManager) buildVolumes(nginxConfigMapName, jwksConfigMapName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: config.VolumeNameNginxConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: nginxConfigMapName,
					},
				},
			},
		},
		{
			Name: config.VolumeNameJWKSData,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: jwksConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  config.ConfigMapKeyJWKS,
							Path: config.ConfigMapKeyJWKS,
						},
					},
				},
			},
		},
	}
}

// buildLabels builds labels for Deployment
func buildLabels(name string) map[string]string {
	return map[string]string{
		config.LabelApp:        name, // Use JWKS resource name as app label value
		config.LabelJWKSConfig: name,
		config.LabelManagedBy:  config.LabelManagedByValue,
	}
}

// buildSelectorLabels builds selector labels for Deployment
func buildSelectorLabels(name string) map[string]string {
	return map[string]string{
		config.LabelApp:        name, // Use JWKS resource name as app label value
		config.LabelJWKSConfig: name,
	}
}

// createDeployment creates a new nginx Deployment spec
func (m *DeploymentManager) createDeployment(
	name, namespace, nginxConfigMapName, jwksConfigMapName, _ string,
	nginxResources *NginxResources,
) *appsv1.Deployment {
	// Get configuration values with defaults
	image := config.DefaultNginxImage
	port := config.DefaultNginxPort
	replicas := int32(config.DefaultNginxReplicas)

	if m.config != nil {
		if m.config.Image != "" {
			image = m.config.Image
		}
		if m.config.Port > 0 {
			port = m.config.Port
		}
		if m.config.Replicas > 0 {
			replicas = m.config.Replicas
		}
	}

	container := m.buildContainerSpec(name, image, port, nginxConfigMapName, jwksConfigMapName, nginxResources)
	volumes := m.buildVolumes(nginxConfigMapName, jwksConfigMapName)
	labels := buildLabels(name)
	selectorLabels := buildSelectorLabels(name)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
					Volumes:    volumes,
				},
			},
		},
	}
}
