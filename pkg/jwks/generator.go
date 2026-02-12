package jwks

import (
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// Generator generates JWKS from certificates
type Generator struct{}

// NewGenerator creates a new JWKS generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateFromCertificate generates JWKS from a PEM-encoded certificate
func (g *Generator) GenerateFromCertificate(certData []byte) (*JWKS, error) {
	cert, err := ParseCertificate(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	if err := ValidateCertificate(cert); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	return g.generateFromCert(cert, certData)
}

// GenerateFromSecret generates JWKS from a Kubernetes Secret
func (g *Generator) GenerateFromSecret(secret *corev1.Secret) (*JWKS, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret is nil")
	}

	cert, err := ParseCertificateFromSecret(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from secret: %w", err)
	}

	certData := secret.Data["tls.crt"]
	return g.generateFromCert(cert, certData)
}

// generateFromCert generates JWKS from a parsed certificate
func (g *Generator) generateFromCert(cert *x509.Certificate, _ []byte) (*JWKS, error) {
	// Extract public key
	publicKey, err := ExtractPublicKey(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key: %w", err)
	}

	// Generate Key ID
	kid, err := GenerateKeyID(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}

	// Format as JWK
	jwk, err := FormatJWK(publicKey, kid, cert)
	if err != nil {
		return nil, fmt.Errorf("failed to format JWK: %w", err)
	}

	return &JWKS{
		Keys: []JWK{*jwk},
	}, nil
}

// MergeJWKS merges old and new JWKS, keeping old keys if needed
func (g *Generator) MergeJWKS(oldJWKS, newJWKS *JWKS) (*JWKS, error) {
	if newJWKS == nil || len(newJWKS.Keys) == 0 {
		return nil, fmt.Errorf("new JWKS is empty")
	}

	if oldJWKS == nil || len(oldJWKS.Keys) == 0 {
		return newJWKS, nil
	}

	// Create a map of existing kids to avoid duplicates
	existingKids := make(map[string]bool)
	for _, key := range oldJWKS.Keys {
		existingKids[key.Kid] = true
	}

	// Start with old keys
	merged := &JWKS{
		Keys: make([]JWK, 0, len(oldJWKS.Keys)+len(newJWKS.Keys)),
	}

	// Add old keys
	merged.Keys = append(merged.Keys, oldJWKS.Keys...)

	// Add new keys (skip if kid already exists)
	for _, key := range newJWKS.Keys {
		if !existingKids[key.Kid] {
			merged.Keys = append(merged.Keys, key)
		}
	}

	return merged, nil
}
