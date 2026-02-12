package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/config"
	"github.com/jwks-operator/jwks-operator/pkg/reconciler"
)

// JWKSReconciler reconciles a JWKS object
type JWKSReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Config     *config.Config
	Logger     *zap.Logger
	Reconciler *reconciler.Reconciler
}

// NewJWKSReconciler creates a new JWKS reconciler
func NewJWKSReconciler(client client.Client, scheme *runtime.Scheme, cfg *config.Config, logger *zap.Logger) *JWKSReconciler {
	return &JWKSReconciler{
		Client:     client,
		Scheme:     scheme,
		Config:     cfg,
		Logger:     logger,
		Reconciler: reconciler.NewReconciler(client, cfg, logger),
	}
}

//+kubebuilder:rbac:groups=example.com,resources=jwks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.com,resources=jwks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.com,resources=jwks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=services,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=create;get;list;update;patch;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;delete;get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *JWKSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the JWKS instance
	jwks := &v1alpha1.JWKS{}
	err := r.Get(ctx, req.NamespacedName, jwks)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// JWKS was deleted, cleanup nginx Deployment if exists
			// Try to find JWKS by checking if nginx deployment exists
			// Deployment name format: <jwks-name>
			logger.Info("JWKS resource not found, checking for cleanup", "name", req.Name, "namespace", req.Namespace)

			// Try to delete nginx deployment (best effort)
			deploymentName := req.Name
			if err := r.Reconciler.Cleanup(ctx, req.Namespace, req.Name); err != nil {
				logger.Info("Cleanup completed", "deployment", deploymentName)
			}

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get JWKS")
		return ctrl.Result{}, err
	}

	// Check if JWKS is being deleted
	if !jwks.DeletionTimestamp.IsZero() {
		logger.Info("JWKS is being deleted, cleaning up resources", "name", jwks.Name)
		return r.handleDeletion(ctx, jwks)
	}

	// Reconcile the JWKS
	if err := r.Reconciler.Reconcile(ctx, jwks); err != nil {
		// Check if error is due to missing Secret
		errMsg := err.Error()
		if contains(errMsg, "failed to get secret") && contains(errMsg, "not found") {
			logger.Info("Secret not found, will retry",
				"secret", jwks.Spec.CertificateSecret,
				"namespace", jwks.Namespace,
			)
			// Requeue after a short interval to check if Secret appears
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		logger.Error(err, "Failed to reconcile JWKS")
		return ctrl.Result{}, err
	}

	// Determine requeue interval based on fast reconciliation logic after restart
	requeueInterval := r.getRequeueInterval(jwks)

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// getReconcileInterval returns reconcile interval from CRD or config default
func (r *JWKSReconciler) getReconcileInterval(jwks *v1alpha1.JWKS) time.Duration {
	if jwks.Spec.ReconcileInterval != "" {
		if d, err := time.ParseDuration(jwks.Spec.ReconcileInterval); err == nil && d > 0 {
			return d
		}
	}
	return r.Config.ReconcileInterval.Duration
}

// getVerificationInterval returns verification interval from CRD or config default
func (r *JWKSReconciler) getVerificationInterval(jwks *v1alpha1.JWKS) time.Duration {
	if jwks.Spec.JWKSVerificationInterval != "" {
		if d, err := time.ParseDuration(jwks.Spec.JWKSVerificationInterval); err == nil && d > 0 {
			return d
		}
	}
	return r.Config.JWKSVerificationInterval.Duration
}

// getRequeueInterval returns the requeue interval with fast reconciliation logic after restart
// First two reconciliations after restart: 10s and 30s, then normal interval
func (r *JWKSReconciler) getRequeueInterval(jwks *v1alpha1.JWKS) time.Duration {
	const (
		fastReconcileAnnotation = "jwks-operator.example.com/fast-reconcile-count"
		firstFastInterval       = 10 * time.Second
		secondFastInterval      = 30 * time.Second
		restartThreshold        = 30 * time.Minute // If last verification was > 30 min ago, consider it a restart
	)

	// Check fast reconcile count from annotation
	fastCount := 0
	if jwks.Annotations != nil {
		fastCountStr := jwks.Annotations[fastReconcileAnnotation]
		if fastCountStr != "" {
			if count, err := strconv.Atoi(fastCountStr); err == nil {
				fastCount = count
			}
		}
	}

	// Check if this is a restart scenario based on last verification time
	// Use verification time instead of update time, as verification happens more frequently
	// Only reset fast count if verification was really long ago (operator restart scenario)
	isRestart := false
	if jwks.Status.JWKSVerified != nil {
		elapsed := time.Since(jwks.Status.JWKSVerified.Time)
		if elapsed > restartThreshold {
			isRestart = true
		}
	} else {
		// No verification time means first reconciliation ever
		isRestart = true
	}

	// If restart detected and fast count is already >= 2, reset it
	// But only reset if we're sure it's a restart (verification was really long ago)
	// This prevents infinite loops when annotation updates fail
	if isRestart && fastCount >= 2 {
		// Only reset if verification was more than threshold ago
		// This ensures we don't reset on normal operation
		fastCount = 0
		// Reset annotation to start fresh (best effort, non-blocking)
		r.updateFastReconcileCountAsync(jwks, 0)
	}

	// Determine interval based on fast reconcile count
	var requeueInterval time.Duration
	switch fastCount {
	case 0:
		// First fast reconciliation: 10 seconds
		requeueInterval = firstFastInterval
		// Update annotation for next reconciliation (best effort, non-blocking)
		r.updateFastReconcileCountAsync(jwks, 1)
	case 1:
		// Second fast reconciliation: 30 seconds
		requeueInterval = secondFastInterval
		// Update annotation to mark fast reconciliations complete (best effort, non-blocking)
		r.updateFastReconcileCountAsync(jwks, 2)
	default:
		// Normal reconciliation: use minimum of reconcile and verification intervals
		// If fastCount is >= 2, we're in normal mode
		reconcileInterval := r.getReconcileInterval(jwks)
		verificationInterval := r.getVerificationInterval(jwks)

		// Use the smaller interval to ensure timely verification
		requeueInterval = reconcileInterval
		if verificationInterval < reconcileInterval {
			requeueInterval = verificationInterval
		}

		// Safety check: if fastCount is stuck (e.g., 1) and we've been in this state for too long,
		// force transition to normal mode to prevent infinite loops
		if fastCount == 1 && jwks.Status.JWKSVerified != nil {
			// If last verification was recent (within last 5 minutes), we're likely stuck
			// Force transition to normal mode
			elapsedSinceVerification := time.Since(jwks.Status.JWKSVerified.Time)
			if elapsedSinceVerification < 5*time.Minute {
				// Recent verification means we're stuck in fast mode, force normal mode
				requeueInterval = verificationInterval
				// Try to update annotation to 2 (best effort)
				r.updateFastReconcileCountAsync(jwks, 2)
			}
		}
	}

	return requeueInterval
}

// updateFastReconcileCountAsync updates the fast reconcile count annotation asynchronously
// This is called to avoid blocking reconciliation and prevent conflicts
func (r *JWKSReconciler) updateFastReconcileCountAsync(jwks *v1alpha1.JWKS, count int) {
	// Run in background goroutine to avoid blocking
	go func() {
		const fastReconcileAnnotation = "jwks-operator.example.com/fast-reconcile-count"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get fresh copy of JWKS to avoid conflicts
		freshJWKS := &v1alpha1.JWKS{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(jwks), freshJWKS); err != nil {
			// If we can't get the resource, silently ignore - it's not critical
			return
		}

		if freshJWKS.Annotations == nil {
			freshJWKS.Annotations = make(map[string]string)
		}

		freshJWKS.Annotations[fastReconcileAnnotation] = fmt.Sprintf("%d", count)

		// Update the annotation (best effort, don't fail if it doesn't work)
		_ = r.Update(ctx, freshJWKS)
		// Silently ignore all errors - annotation update is not critical
		// Conflicts are expected when multiple reconciliations run concurrently
	}()
}

// handleDeletion handles deletion of JWKS resource
func (r *JWKSReconciler) handleDeletion(ctx context.Context, jwks *v1alpha1.JWKS) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Cleanup nginx resources (Service and Deployment)
	if err := r.Reconciler.Cleanup(ctx, jwks.Namespace, jwks.Name); err != nil {
		logger.Error(err, "Failed to cleanup nginx resources")
		// Continue with deletion even if cleanup fails
	}

	// Note: ConfigMaps are not deleted by default (cleanupOnDelete=false)
	// This allows manual cleanup or reuse of ConfigMaps

	return ctrl.Result{}, nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// SetupWithManager sets up the controller with the Manager
func (r *JWKSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.JWKS{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
