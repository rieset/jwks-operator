package reconciler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/config"
	"github.com/jwks-operator/jwks-operator/pkg/configmap"
	"github.com/jwks-operator/jwks-operator/pkg/jwks"
	"github.com/jwks-operator/jwks-operator/pkg/metrics"
	"github.com/jwks-operator/jwks-operator/pkg/nginx"
	"github.com/jwks-operator/jwks-operator/pkg/verification"
)

// ReconciliationLoop executes the reconciliation process
type ReconciliationLoop struct {
	client           client.Client
	jwksGenerator    *jwks.Generator
	configMapManager *configmap.Manager
	nginxManager     *nginx.Manager
	statusUpdater    *StatusUpdater
	verifier         *verification.Verifier
	config           *config.Config
	logger           *zap.Logger
}

// NewReconciliationLoop creates a new reconciliation loop
func NewReconciliationLoop(
	client client.Client,
	jwksGenerator *jwks.Generator,
	configMapManager *configmap.Manager,
	nginxManager *nginx.Manager,
	statusUpdater *StatusUpdater,
	cfg *config.Config,
	logger *zap.Logger,
) *ReconciliationLoop {
	return &ReconciliationLoop{
		client:           client,
		jwksGenerator:    jwksGenerator,
		configMapManager: configMapManager,
		nginxManager:     nginxManager,
		statusUpdater:    statusUpdater,
		verifier:         verification.NewVerifier(&cfg.Verification),
		config:           cfg,
		logger:           logger,
	}
}

// Execute executes the reconciliation loop
func (l *ReconciliationLoop) Execute(ctx context.Context, jwks *v1alpha1.JWKS) error {
	startTime := time.Now()
	result := metrics.ResultSuccess

	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordReconcile(result, duration)
	}()

	if jwks == nil {
		result = metrics.ResultError
		metrics.RecordError("jwks_nil")
		return fmt.Errorf("JWKS is nil")
	}

	// Phase 1: Get Secret with certificate
	secret, err := l.phase1GetSecret(ctx, jwks)
	if err != nil {
		result = metrics.ResultError
		metrics.RecordError("secret_not_found")
		l.statusUpdater.SetNotReady(jwks, "SecretNotFound", fmt.Sprintf("Failed to get secret: %v", err))
		return err
	}

	// Phase 2: Generate JWKS from certificate
	newJWKS, err := l.phase2GenerateJWKS(secret)
	if err != nil {
		result = metrics.ResultError
		metrics.RecordError("jwks_generation_failed")
		l.statusUpdater.SetNotReady(jwks, "JWKSGenerationFailed", fmt.Sprintf("Failed to generate JWKS: %v", err))
		return err
	}

	// Phase 3: Ensure JWKS ConfigMap exists and update with JWKS
	if err := l.phase3UpdateConfigMap(ctx, jwks, newJWKS); err != nil {
		result = metrics.ResultError
		metrics.RecordError("configmap_update_failed")
		l.statusUpdater.SetNotReady(jwks, "ConfigMapUpdateFailed", fmt.Sprintf("Failed to update ConfigMap: %v", err))
		return err
	}

	// Phase 4: Ensure nginx ConfigMap exists and update if configured
	if err := l.phase4UpdateNginxConfig(ctx, jwks); err != nil {
		result = metrics.ResultError
		metrics.RecordError("nginx_config_update_failed")
		l.statusUpdater.SetNotReady(jwks, "NginxConfigUpdateFailed", fmt.Sprintf("Failed to update nginx config: %v", err))
		return err
	}

	// Phase 5: Ensure nginx Deployment exists
	if err := l.phase5EnsureNginxDeployment(ctx, jwks); err != nil {
		result = metrics.ResultError
		metrics.RecordError("nginx_deployment_failed")
		l.statusUpdater.SetNotReady(jwks, "NginxDeploymentFailed", fmt.Sprintf("Failed to ensure nginx deployment: %v", err))
		return err
	}

	// Phase 6: Ensure nginx Service exists
	if err := l.phase6EnsureNginxService(ctx, jwks); err != nil {
		result = metrics.ResultError
		metrics.RecordError("nginx_service_failed")
		l.statusUpdater.SetNotReady(jwks, "NginxServiceFailed", fmt.Sprintf("Failed to ensure nginx service: %v", err))
		return err
	}

	// Phase 7: Verify JWKS from nginx (periodic verification)
	// Verification errors are non-critical, continue even if verification fails
	_ = l.phase7VerifyJWKS(ctx, jwks, secret)

	// Update status
	l.statusUpdater.UpdateLastKeyID(jwks, newJWKS.Keys[0].Kid)
	l.statusUpdater.UpdateKeyCount(jwks, len(newJWKS.Keys))
	l.statusUpdater.SetReady(jwks, "JWKS successfully updated")

	return nil
}

// shouldReconcile determines if reconciliation is needed
func (l *ReconciliationLoop) shouldReconcile(ctx context.Context, jwks *v1alpha1.JWKS) bool {
	// Always reconcile if spec has changed (generation != observedGeneration)
	// Check observedGeneration from Ready condition
	observedGeneration := int64(0)
	for _, condition := range jwks.Status.Conditions {
		if condition.Type == "Ready" {
			observedGeneration = condition.ObservedGeneration
			break
		}
	}

	// If generation changed, spec was modified - always reconcile
	if jwks.Generation != observedGeneration {
		return true
	}

	// Always reconcile if LastUpdateTime is nil (first reconciliation)
	if jwks.Status.LastUpdateTime == nil {
		return true
	}

	// Check if nginx resources need to be created/updated
	// This handles cases when operator restarts or is updated
	// and resources might have been deleted or don't exist
	if jwks.Spec.NginxConfigMapName != "" {
		// Check if Deployment exists - if not, we need full reconciliation
		deploymentName := jwks.Name
		deployment := &appsv1.Deployment{}
		deploymentKey := types.NamespacedName{Namespace: jwks.Namespace, Name: deploymentName}
		if err := l.client.Get(ctx, deploymentKey, deployment); err != nil {
			if apierrors.IsNotFound(err) {
				// Deployment doesn't exist - need full reconciliation to create it
				return true
			}
			// Other error - log but don't fail, will retry
			l.logger.Debug("failed to check deployment existence, will retry",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.Error(err),
			)
		}

		// Check if nginx ConfigMap exists - if not, we need full reconciliation
		nginxConfigMap := &corev1.ConfigMap{}
		nginxKey := types.NamespacedName{Namespace: jwks.Namespace, Name: jwks.Spec.NginxConfigMapName}
		if err := l.client.Get(ctx, nginxKey, nginxConfigMap); err != nil {
			if apierrors.IsNotFound(err) {
				// ConfigMap doesn't exist - need full reconciliation to create it
				return true
			}
			// Other error - log but don't fail, will retry
			l.logger.Debug("failed to check nginx ConfigMap existence, will retry",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.String("nginxConfigMap", jwks.Spec.NginxConfigMapName),
				zap.Error(err),
			)
		}

		// Check if JWKS ConfigMap exists - if not, we need full reconciliation
		jwksConfigMap := &corev1.ConfigMap{}
		jwksKey := types.NamespacedName{Namespace: jwks.Namespace, Name: jwks.Spec.ConfigMapName}
		if err := l.client.Get(ctx, jwksKey, jwksConfigMap); err != nil {
			if apierrors.IsNotFound(err) {
				// ConfigMap doesn't exist - need full reconciliation to create it
				return true
			}
			// Other error - log but don't fail, will retry
			l.logger.Debug("failed to check JWKS ConfigMap existence, will retry",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.String("configMap", jwks.Spec.ConfigMapName),
				zap.Error(err),
			)
		}
	}

	// Check if enough time has passed since last update
	elapsed := time.Since(jwks.Status.LastUpdateTime.Time)
	updateInterval := l.getJWKSUpdateInterval(jwks)
	return elapsed >= updateInterval
}
