package reconciler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/config"
	"github.com/jwks-operator/jwks-operator/pkg/configmap"
	"github.com/jwks-operator/jwks-operator/pkg/jwks"
	"github.com/jwks-operator/jwks-operator/pkg/metrics"
	"github.com/jwks-operator/jwks-operator/pkg/utils"
)

// phase1GetSecret gets the Secret with certificate
func (l *ReconciliationLoop) phase1GetSecret(ctx context.Context, jwks *v1alpha1.JWKS) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: jwks.Namespace,
		Name:      jwks.Spec.CertificateSecret,
	}

	err := l.client.Get(ctx, key, secret)
	if err != nil {
		l.logger.Error("failed to get secret",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.String("secret", jwks.Spec.CertificateSecret),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get secret %s: %w", jwks.Spec.CertificateSecret, err)
	}

	l.logger.Debug("secret retrieved successfully",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("secret", jwks.Spec.CertificateSecret),
	)

	return secret, nil
}

// phase2GenerateJWKS generates JWKS from certificate
func (l *ReconciliationLoop) phase2GenerateJWKS(secret *corev1.Secret) (*jwks.JWKS, error) {
	newJWKS, err := l.jwksGenerator.GenerateFromSecret(secret)
	if err != nil {
		l.logger.Error("failed to generate JWKS from secret",
			zap.Error(err),
		)
		metrics.RecordJWKSGeneration(metrics.ResultError)
		return nil, fmt.Errorf("failed to generate JWKS: %w", err)
	}

	if len(newJWKS.Keys) == 0 {
		l.logger.Error("generated JWKS has no keys")
		metrics.RecordJWKSGeneration(metrics.ResultError)
		return nil, fmt.Errorf("generated JWKS has no keys")
	}

	metrics.RecordJWKSGeneration(metrics.ResultSuccess)
	l.logger.Debug("JWKS generated successfully",
		zap.Int("keyCount", len(newJWKS.Keys)),
		zap.String("firstKeyID", newJWKS.Keys[0].Kid),
	)

	return newJWKS, nil
}

// phase3UpdateConfigMap ensures JWKS ConfigMap exists and updates it
func (l *ReconciliationLoop) phase3UpdateConfigMap(ctx context.Context, jwks *v1alpha1.JWKS, newJWKS *jwks.JWKS) error {
	l.logger.Debug("updating JWKS ConfigMap",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("configMap", jwks.Spec.ConfigMapName),
	)

	// Check if ConfigMap exists, recreate if deleted
	if err := l.ensureJWKSConfigMap(ctx, jwks, newJWKS); err != nil {
		l.logger.Error("failed to ensure JWKS ConfigMap",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.String("configMap", jwks.Spec.ConfigMapName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to ensure ConfigMap: %w", err)
	}

	updateStrategy := l.getUpdateStrategy(jwks)
	keepOldKeys := l.shouldKeepOldKeys(jwks)

	l.logger.Debug("applying update strategy",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("strategy", updateStrategy),
		zap.Bool("keepOldKeys", keepOldKeys),
	)

	strategy := configmap.NewUpdateStrategy(l.configMapManager)
	if err := strategy.Apply(ctx, jwks.Namespace, jwks.Spec.ConfigMapName, newJWKS, updateStrategy, keepOldKeys); err != nil {
		l.logger.Error("failed to update ConfigMap",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.String("configMap", jwks.Spec.ConfigMapName),
			zap.Error(err),
		)
		metrics.RecordConfigMapUpdate("jwks", metrics.ResultError)
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	metrics.RecordConfigMapUpdate("jwks", metrics.ResultSuccess)
	l.logger.Info("JWKS ConfigMap updated successfully",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("configMap", jwks.Spec.ConfigMapName),
	)

	return nil
}

// phase4UpdateNginxConfig ensures nginx ConfigMap exists and updates it
func (l *ReconciliationLoop) phase4UpdateNginxConfig(ctx context.Context, jwks *v1alpha1.JWKS) error {
	if jwks.Spec.NginxConfigMapName == "" {
		return nil // Nginx not configured
	}

	l.logger.Debug("updating nginx ConfigMap",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("nginxConfigMap", jwks.Spec.NginxConfigMapName),
	)

	endpoint := l.getEndpoint(jwks)
	if err := l.ensureNginxConfigMap(ctx, jwks, endpoint); err != nil {
		l.logger.Error("failed to ensure nginx ConfigMap",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.String("nginxConfigMap", jwks.Spec.NginxConfigMapName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to ensure nginx ConfigMap: %w", err)
	}

	if err := l.nginxManager.UpdateConfig(ctx, jwks.Namespace, jwks.Spec.NginxConfigMapName, jwks.Spec.ConfigMapName, endpoint); err != nil {
		l.logger.Error("failed to update nginx config",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.String("nginxConfigMap", jwks.Spec.NginxConfigMapName),
			zap.Error(err),
		)
		metrics.RecordConfigMapUpdate("nginx", metrics.ResultError)
		metrics.RecordNginxOperation("config", metrics.ResultError)
		return fmt.Errorf("failed to update nginx config: %w", err)
	}

	metrics.RecordConfigMapUpdate("nginx", metrics.ResultSuccess)
	metrics.RecordNginxOperation("config", metrics.ResultSuccess)
	l.logger.Info("nginx ConfigMap updated successfully",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("nginxConfigMap", jwks.Spec.NginxConfigMapName),
	)

	l.statusUpdater.UpdateNginxConfigUpdated(jwks)
	return nil
}

// phase5EnsureNginxDeployment ensures nginx Deployment exists
func (l *ReconciliationLoop) phase5EnsureNginxDeployment(ctx context.Context, jwks *v1alpha1.JWKS) error {
	if jwks.Spec.NginxConfigMapName == "" {
		return nil // Nginx not configured
	}

	l.logger.Debug("ensuring nginx Deployment",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
	)

	endpoint := l.getEndpoint(jwks)
	if err := l.nginxManager.EnsureDeployment(
		ctx,
		jwks.Namespace,
		jwks.Name,
		jwks.Spec.NginxConfigMapName,
		jwks.Spec.ConfigMapName,
		endpoint,
		&l.config.Nginx.Resources,
	); err != nil {
		l.logger.Error("failed to ensure nginx deployment",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Error(err),
		)
		metrics.RecordNginxOperation("deployment", metrics.ResultError)
		return fmt.Errorf("failed to ensure nginx deployment: %w", err)
	}

	metrics.RecordNginxOperation("deployment", metrics.ResultSuccess)
	l.logger.Info("nginx Deployment ensured successfully",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
	)

	return nil
}

// phase6EnsureNginxService ensures nginx Service exists
func (l *ReconciliationLoop) phase6EnsureNginxService(ctx context.Context, jwks *v1alpha1.JWKS) error {
	if jwks.Spec.NginxConfigMapName == "" {
		return nil // Nginx not configured
	}

	l.logger.Debug("ensuring nginx Service",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
	)

	if err := l.nginxManager.EnsureService(ctx, jwks.Namespace, jwks.Name); err != nil {
		l.logger.Error("failed to ensure nginx service",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Error(err),
		)
		metrics.RecordNginxOperation("service", metrics.ResultError)
		return fmt.Errorf("failed to ensure nginx service: %w", err)
	}

	metrics.RecordNginxOperation("service", metrics.ResultSuccess)
	l.logger.Info("nginx Service ensured successfully",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
	)

	return nil
}

// phase7VerifyJWKS verifies JWKS from nginx (periodic verification)
func (l *ReconciliationLoop) phase7VerifyJWKS(ctx context.Context, jwks *v1alpha1.JWKS, secret *corev1.Secret) error {
	if jwks.Spec.NginxConfigMapName == "" {
		return nil // Nginx not configured
	}

	// Check if this is first fast reconciliation after restart (force verification)
	forceVerification := false
	const fastReconcileAnnotation = "jwks-operator.example.com/fast-reconcile-count"
	if jwks.Annotations != nil {
		fastCountStr := jwks.Annotations[fastReconcileAnnotation]
		if fastCountStr == "" || fastCountStr == "0" {
			// First fast reconciliation after restart - force verification
			forceVerification = true
		}
	} else {
		// No annotation means first reconciliation ever - force verification
		forceVerification = true
	}

	if !l.shouldVerifyJWKS(jwks, forceVerification) {
		return nil // Not time to verify yet
	}

	// Check if Service has ready endpoints before verification
	if err := l.waitForServiceEndpoints(ctx, jwks.Namespace, jwks.Name); err != nil {
		l.logger.Warn("Service endpoints not ready, skipping verification",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.String("service", jwks.Name),
			zap.Error(err),
		)
		// Don't return error - will retry on next reconciliation
		return nil
	}

	l.logger.Debug("starting JWKS verification",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
		zap.String("service", jwks.Name),
	)

	// Use a context with timeout for verification
	contextTimeout := config.DefaultVerificationContextTimeout
	if l.config.Verification.ContextTimeout > 0 {
		contextTimeout = l.config.Verification.ContextTimeout
	}
	verifyCtx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	// Retry verification a few times in case nginx is still starting
	retryCount := config.DefaultVerificationRetryCount
	retryDelay := config.DefaultVerificationRetryDelay
	if l.config.Verification.RetryCount > 0 {
		retryCount = l.config.Verification.RetryCount
	}
	if l.config.Verification.RetryDelay > 0 {
		retryDelay = l.config.Verification.RetryDelay
	}

	var verifyErr error
	attemptCount := 0

	retryConfig := utils.RetryConfig{
		MaxAttempts: retryCount,
		Delay:       retryDelay,
		OnRetry: func(attempt int, _ error) {
			l.logger.Debug("retrying JWKS verification",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.Int("attempt", attempt+1),
				zap.Int("maxAttempts", retryCount),
			)
		},
	}

	verifyErr = utils.RetryWithDelay(verifyCtx, retryConfig, func() error {
		attemptCount++
		err := l.verifier.VerifyJWKSFromNginx(verifyCtx, jwks.Namespace, jwks.Name, secret)
		if err != nil {
			l.logger.Debug("JWKS verification attempt failed",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.Int("attempt", attemptCount),
				zap.Int("maxAttempts", retryCount),
				zap.Error(err),
			)
			return err
		}

		// Success - will log after retry loop completes
		return nil
	})

	if verifyErr != nil {
		l.logger.Warn("JWKS verification failed after all retry attempts",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Int("attempts", retryCount),
			zap.Error(verifyErr),
		)
		metrics.RecordJWKSVerification(metrics.ResultError)
		l.statusUpdater.SetNotReady(jwks, "JWKSVerificationFailed", fmt.Sprintf("Failed to verify JWKS from nginx: %v", verifyErr))
		// Don't return error - verification failure is a warning, not a critical error
		// The operator will retry verification on next reconciliation
		return nil
	}

	// Log success message only if retries were needed (to reduce log frequency)
	// Normal successful verifications are logged at Debug level to avoid spam
	if attemptCount > 1 {
		l.logger.Info("JWKS verification succeeded after retries",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Int("attempts", attemptCount),
		)
	} else {
		// First attempt succeeded - log at Debug to reduce frequency
		l.logger.Debug("JWKS verification succeeded",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
		)
	}

	metrics.RecordJWKSVerification(metrics.ResultSuccess)
	l.statusUpdater.UpdateJWKSVerified(jwks)
	return nil
}

// ensureJWKSConfigMap checks if JWKS ConfigMap exists, recreates if deleted
func (l *ReconciliationLoop) ensureJWKSConfigMap(ctx context.Context, jwks *v1alpha1.JWKS, jwksData *jwks.JWKS) error {
	exists, err := utils.EnsureConfigMapExists(ctx, l.client, jwks.Namespace, jwks.Spec.ConfigMapName)
	if err != nil {
		return err
	}

	if exists {
		// ConfigMap exists, nothing to do
		return nil
	}

	// ConfigMap was deleted, recreate it
	return l.configMapManager.CreateConfigMap(ctx, jwks.Namespace, jwks.Spec.ConfigMapName, jwksData)
}

// ensureNginxConfigMap checks if nginx ConfigMap exists, recreates if deleted
func (l *ReconciliationLoop) ensureNginxConfigMap(ctx context.Context, jwks *v1alpha1.JWKS, endpoint string) error {
	exists, err := utils.EnsureConfigMapExists(ctx, l.client, jwks.Namespace, jwks.Spec.NginxConfigMapName)
	if err != nil {
		return fmt.Errorf("failed to check nginx ConfigMap: %w", err)
	}

	if exists {
		// ConfigMap exists, nothing to do
		return nil
	}

	// ConfigMap was deleted, recreate it by calling UpdateConfig which will create it
	return l.nginxManager.UpdateConfig(ctx, jwks.Namespace, jwks.Spec.NginxConfigMapName, jwks.Spec.ConfigMapName, endpoint)
}

// waitForServiceEndpoints waits for Service to have ready endpoints
// Uses Pods directly instead of Endpoints to avoid deprecation warnings
func (l *ReconciliationLoop) waitForServiceEndpoints(ctx context.Context, namespace, serviceName string) error {
	// First, check if Service exists and get its selectors
	service := &corev1.Service{}
	serviceKey := types.NamespacedName{Namespace: namespace, Name: serviceName}
	if err := l.client.Get(ctx, serviceKey, service); err != nil {
		return fmt.Errorf("service %s/%s not found: %w", namespace, serviceName, err)
	}

	selectors := service.Spec.Selector
	if len(selectors) == 0 {
		return fmt.Errorf("service %s/%s has no selectors", namespace, serviceName)
	}

	// Try to find ready pods matching service selectors with retry
	maxAttempts := 5
	delay := 2 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check if there are any ready pods matching the service selectors
		podList := &corev1.PodList{}
		listOpts := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingLabels(selectors),
		}
		if err := l.client.List(ctx, podList, listOpts...); err == nil {
			// Check if at least one pod is ready
			for _, pod := range podList.Items {
				if pod.Status.Phase == corev1.PodRunning {
					for _, condition := range pod.Status.Conditions {
						if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
							// At least one pod is ready
							return nil
						}
					}
				}
			}

			// On last attempt, log detailed diagnostic information
			if attempt == maxAttempts-1 {
				podCount := len(podList.Items)
				readyPods := 0
				runningPods := 0
				for _, pod := range podList.Items {
					if pod.Status.Phase == corev1.PodRunning {
						runningPods++
						for _, condition := range pod.Status.Conditions {
							if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
								readyPods++
								break
							}
						}
					}
				}

				// Check if Deployment exists and its status
				deploymentName := serviceName
				deployment := &appsv1.Deployment{}
				deploymentKey := types.NamespacedName{Namespace: namespace, Name: deploymentName}
				deploymentErr := l.client.Get(ctx, deploymentKey, deployment)
				if deploymentErr != nil {
					// Try with nginx- prefix
					deploymentKey.Name = "nginx-" + serviceName
					deploymentErr = l.client.Get(ctx, deploymentKey, deployment)
				}

				var deploymentInfo string
				if deploymentErr == nil {
					replicas := int32(0)
					readyReplicas := int32(0)
					availableReplicas := int32(0)
					if deployment.Spec.Replicas != nil {
						replicas = *deployment.Spec.Replicas
					}
					if deployment.Status.ReadyReplicas > 0 {
						readyReplicas = deployment.Status.ReadyReplicas
					}
					if deployment.Status.AvailableReplicas > 0 {
						availableReplicas = deployment.Status.AvailableReplicas
					}
					deploymentLabels := deployment.Spec.Template.Labels
					deploymentInfo = fmt.Sprintf("deployment %s/%s exists: replicas=%d, ready=%d, available=%d, podLabels=%v",
						namespace, deploymentKey.Name, replicas, readyReplicas, availableReplicas, deploymentLabels)
				} else {
					deploymentInfo = fmt.Sprintf("deployment %s/%s or nginx-%s/%s not found", namespace, serviceName, namespace, serviceName)
				}

				return fmt.Errorf("service %s/%s has no ready pods after %d attempts: found %d pods matching selectors %v, %d running, %d ready; %s",
					namespace, serviceName, maxAttempts, podCount, selectors, runningPods, readyPods, deploymentInfo)
			}
		}

		if attempt < maxAttempts-1 {
			// Wait before next attempt
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return fmt.Errorf("service %s/%s has no ready pods after %d attempts (selectors: %v)", namespace, serviceName, maxAttempts, selectors)
}
