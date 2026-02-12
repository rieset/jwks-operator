package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JWKSSpec defines the desired state of JWKS
type JWKSSpec struct {
	// CertificateSecret is the name of the Secret containing the JWT certificate
	// +kubebuilder:validation:Required
	CertificateSecret string `json:"certificateSecret"`

	// ConfigMapName is the name of the ConfigMap to store JWKS data
	// +kubebuilder:validation:Required
	ConfigMapName string `json:"configMapName"`

	// NginxConfigMapName is the name of the ConfigMap for nginx configuration
	// +optional
	NginxConfigMapName string `json:"nginxConfigMapName,omitempty"`

	// Endpoint is the HTTP endpoint path for JWKS
	// JWKS will be available at both "/" and "/jwks.json" paths
	// This field is kept for backward compatibility but is not used in nginx config generation
	// +kubebuilder:default="/jwks.json"
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// UpdateStrategy defines how to update JWKS when certificate rotates
	// +kubebuilder:validation:Enum=rolling;immediate
	// +kubebuilder:default=rolling
	// +optional
	UpdateStrategy string `json:"updateStrategy,omitempty"`

	// KeepOldKeys determines if old keys should be kept during rotation
	// +kubebuilder:default=true
	// +optional
	KeepOldKeys bool `json:"keepOldKeys,omitempty"`

	// OldKeysTTL is the time to keep old keys after rotation
	// Format: Go duration (e.g., "720h" for 30 days)
	// +kubebuilder:default="720h"
	// +optional
	OldKeysTTL string `json:"oldKeysTTL,omitempty"`

	// ReconcileInterval is the interval between reconciliations
	// Format: Go duration (e.g., "5m", "1h")
	// If not specified, uses operator default from config.yaml
	// +optional
	ReconcileInterval string `json:"reconcileInterval,omitempty"`

	// JWKSUpdateInterval is the interval for checking JWKS updates
	// Format: Go duration (e.g., "6h", "24h")
	// If not specified, uses operator default from config.yaml
	// +optional
	JWKSUpdateInterval string `json:"jwksUpdateInterval,omitempty"`

	// JWKSVerificationInterval is the interval for verifying JWKS from nginx
	// Format: Go duration (e.g., "5m", "10m")
	// If not specified, uses operator default from config.yaml (5 minutes)
	// +optional
	JWKSVerificationInterval string `json:"jwksVerificationInterval,omitempty"`
}

// JWKSStatus defines the observed state of JWKS
type JWKSStatus struct {
	// Conditions represent the latest available observations of the JWKS's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdateTime is the timestamp of the last JWKS update
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastKeyID is the Key ID (kid) of the last generated key
	// +optional
	LastKeyID string `json:"lastKeyID,omitempty"`

	// KeyCount is the number of keys in the current JWKS
	// +optional
	KeyCount int `json:"keyCount,omitempty"`

	// NginxConfigUpdated is the timestamp when nginx config was last updated
	// +optional
	NginxConfigUpdated *metav1.Time `json:"nginxConfigUpdated,omitempty"`

	// JWKSVerified is the timestamp when JWKS was last verified from nginx
	// +optional
	JWKSVerified *metav1.Time `json:"jwksVerified,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=jwks,scope=Namespaced,shortName=jwks
//+kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".spec.certificateSecret"
//+kubebuilder:printcolumn:name="ConfigMap",type="string",JSONPath=".spec.configMapName"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// JWKS is the Schema for the jwks API
type JWKS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JWKSSpec   `json:"spec,omitempty"`
	Status JWKSStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JWKSList contains a list of JWKS
type JWKSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JWKS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JWKS{}, &JWKSList{})
}
