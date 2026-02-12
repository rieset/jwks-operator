package reconciler

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
)

// StatusUpdater updates the status of JWKS resources
type StatusUpdater struct {
	client client.Client
}

// NewStatusUpdater creates a new status updater
func NewStatusUpdater(client client.Client) *StatusUpdater {
	return &StatusUpdater{
		client: client,
	}
}

// UpdateStatus updates the status of a JWKS resource
func (u *StatusUpdater) UpdateStatus(ctx context.Context, jwks *v1alpha1.JWKS, status *v1alpha1.JWKSStatus) error {
	if jwks == nil {
		return fmt.Errorf("JWKS is nil")
	}

	jwks.Status = *status
	return u.client.Status().Update(ctx, jwks)
}

// SetCondition sets a condition on JWKS
func (u *StatusUpdater) SetCondition(jwks *v1alpha1.JWKS, conditionType string, status metav1.ConditionStatus, reason, message string) {
	if jwks == nil {
		return
	}

	now := metav1.Now()
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: jwks.Generation,
	}

	// Find and update existing condition or add new one
	found := false
	for i, c := range jwks.Status.Conditions {
		if c.Type == conditionType {
			jwks.Status.Conditions[i] = condition
			found = true
			break
		}
	}

	if !found {
		jwks.Status.Conditions = append(jwks.Status.Conditions, condition)
	}
}

// SetReady sets the Ready condition to true
func (u *StatusUpdater) SetReady(jwks *v1alpha1.JWKS, message string) {
	u.SetCondition(jwks, "Ready", metav1.ConditionTrue, "Reconciled", message)
	now := metav1.Now()
	jwks.Status.LastUpdateTime = &now
}

// SetNotReady sets the Ready condition to false
func (u *StatusUpdater) SetNotReady(jwks *v1alpha1.JWKS, reason, message string) {
	u.SetCondition(jwks, "Ready", metav1.ConditionFalse, reason, message)
}

// UpdateLastKeyID updates the last key ID in status
func (u *StatusUpdater) UpdateLastKeyID(jwks *v1alpha1.JWKS, kid string) {
	if jwks == nil {
		return
	}
	jwks.Status.LastKeyID = kid
}

// UpdateKeyCount updates the key count in status
func (u *StatusUpdater) UpdateKeyCount(jwks *v1alpha1.JWKS, count int) {
	if jwks == nil {
		return
	}
	jwks.Status.KeyCount = count
}

// UpdateNginxConfigUpdated updates the nginx config update time
func (u *StatusUpdater) UpdateNginxConfigUpdated(jwks *v1alpha1.JWKS) {
	if jwks == nil {
		return
	}
	now := metav1.Now()
	jwks.Status.NginxConfigUpdated = &now
}

// UpdateJWKSVerified updates the JWKS verification time
func (u *StatusUpdater) UpdateJWKSVerified(jwks *v1alpha1.JWKS) {
	if jwks == nil {
		return
	}
	now := metav1.Now()
	jwks.Status.JWKSVerified = &now
}
