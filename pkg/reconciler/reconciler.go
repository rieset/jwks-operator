package reconciler

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/config"
	"github.com/jwks-operator/jwks-operator/pkg/configmap"
	"github.com/jwks-operator/jwks-operator/pkg/jwks"
	"github.com/jwks-operator/jwks-operator/pkg/nginx"
)

// Reconciler reconciles JWKS resources
type Reconciler struct {
	client             client.Client
	jwksGenerator      *jwks.Generator
	configMapManager   *configmap.Manager
	nginxManager       *nginx.Manager
	statusUpdater      *StatusUpdater
	reconciliationLoop *ReconciliationLoop
	config             *config.Config
	logger             *zap.Logger
}

// NewReconciler creates a new reconciler
func NewReconciler(
	client client.Client,
	cfg *config.Config,
	logger *zap.Logger,
) *Reconciler {
	jwksGenerator := jwks.NewGenerator()
	configMapManager := configmap.NewManager(client)
	nginxManager := nginx.NewManager(client, &cfg.Nginx)
	statusUpdater := NewStatusUpdater(client)

	reconciliationLoop := NewReconciliationLoop(
		client,
		jwksGenerator,
		configMapManager,
		nginxManager,
		statusUpdater,
		cfg,
		logger,
	)

	return &Reconciler{
		client:             client,
		jwksGenerator:      jwksGenerator,
		configMapManager:   configMapManager,
		nginxManager:       nginxManager,
		statusUpdater:      statusUpdater,
		reconciliationLoop: reconciliationLoop,
		config:             cfg,
		logger:             logger,
	}
}

// Reconcile reconciles a JWKS resource
func (r *Reconciler) Reconcile(ctx context.Context, jwks *v1alpha1.JWKS) error {
	if jwks == nil {
		return fmt.Errorf("JWKS is nil")
	}

	// Check if full reconciliation is needed (JWKS update)
	needsReconciliation := r.reconciliationLoop.shouldReconcile(ctx, jwks)

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

	// Check if verification is needed (independent of reconciliation)
	needsVerification := r.reconciliationLoop.shouldVerifyJWKS(jwks, forceVerification)

	// If neither reconciliation nor verification is needed, skip
	if !needsReconciliation && !needsVerification {
		r.logger.Debug("skipping reconciliation and verification - too soon since last update/verification",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
		)
		// Return nil - controller will handle RequeueAfter based on getRequeueInterval
		return nil
	}

	// If only verification is needed, perform verification only
	if !needsReconciliation && needsVerification {
		r.logger.Debug("performing JWKS verification only",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
		)

		// Get Secret for verification
		secret, err := r.getSecretForVerification(ctx, jwks)
		if err != nil {
			r.logger.Warn("failed to get secret for verification, skipping",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.Error(err),
			)
			return nil // Don't fail on verification-only run
		}

		// Perform verification only
		_ = r.performVerificationOnly(ctx, jwks, secret)

		// Update status to reflect verification attempt
		if err := r.statusUpdater.UpdateStatus(ctx, jwks, &jwks.Status); err != nil {
			r.logger.Warn("failed to update status after verification",
				zap.String("namespace", jwks.Namespace),
				zap.String("name", jwks.Name),
				zap.Error(err),
			)
		}

		return nil
	}

	// Full reconciliation is needed
	// Execute reconciliation loop (includes verification if needed)
	if err := r.reconciliationLoop.Execute(ctx, jwks); err != nil {
		r.logger.Error("reconciliation failed",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Error(err),
		)
		return err
	}

	// Update status
	if err := r.statusUpdater.UpdateStatus(ctx, jwks, &jwks.Status); err != nil {
		r.logger.Error("failed to update status",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Log single success message for reconciliation
	r.logger.Info("reconciliation completed successfully",
		zap.String("namespace", jwks.Namespace),
		zap.String("name", jwks.Name),
	)

	return nil
}

// Cleanup cleans up resources associated with JWKS
func (r *Reconciler) Cleanup(ctx context.Context, namespace, jwksName string) error {
	r.logger.Info("cleaning up resources for JWKS",
		zap.String("namespace", namespace),
		zap.String("name", jwksName),
	)

	// Delete nginx Service
	if err := r.nginxManager.DeleteService(ctx, namespace, jwksName); err != nil {
		r.logger.Warn("failed to delete nginx service",
			zap.String("namespace", namespace),
			zap.String("name", jwksName),
			zap.Error(err),
		)
		// Continue cleanup even if service deletion fails
	}

	// Delete nginx Deployment
	if err := r.nginxManager.DeleteDeployment(ctx, namespace, jwksName); err != nil {
		r.logger.Warn("failed to delete nginx deployment",
			zap.String("namespace", namespace),
			zap.String("name", jwksName),
			zap.Error(err),
		)
		// Continue cleanup even if deployment deletion fails
	}

	r.logger.Info("cleanup completed",
		zap.String("namespace", namespace),
		zap.String("name", jwksName),
	)

	return nil
}

// getSecretForVerification gets Secret for verification purposes
func (r *Reconciler) getSecretForVerification(ctx context.Context, jwks *v1alpha1.JWKS) (*corev1.Secret, error) {
	return r.reconciliationLoop.phase1GetSecret(ctx, jwks)
}

// performVerificationOnly performs only JWKS verification without full reconciliation
func (r *Reconciler) performVerificationOnly(ctx context.Context, jwks *v1alpha1.JWKS, secret *corev1.Secret) error {
	// Check if Nginx is configured
	if jwks.Spec.NginxConfigMapName == "" {
		return nil // Nginx not configured, skip verification
	}

	// Before verification, ensure Service exists
	// This is important because performVerificationOnly can be called before full reconciliation
	if err := r.nginxManager.EnsureService(ctx, jwks.Namespace, jwks.Name); err != nil {
		r.logger.Warn("failed to ensure service before verification, skipping",
			zap.String("namespace", jwks.Namespace),
			zap.String("name", jwks.Name),
			zap.Error(err),
		)
		// Don't return error - service will be created on next full reconciliation
		return nil
	}

	return r.reconciliationLoop.phase7VerifyJWKS(ctx, jwks, secret)
}
