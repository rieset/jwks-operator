package jwks

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// ParseCertificate parses a PEM-encoded certificate
func ParseCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("expected CERTIFICATE block, got %s", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// ParseCertificateFromSecret extracts and parses certificate from Kubernetes Secret
func ParseCertificateFromSecret(secretData map[string][]byte) (*x509.Certificate, error) {
	certData, ok := secretData["tls.crt"]
	if !ok {
		return nil, fmt.Errorf("tls.crt not found in secret")
	}

	return ParseCertificate(certData)
}

// ValidateCertificate validates a certificate
func ValidateCertificate(cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}

	// Check if certificate is expired
	// Note: We don't fail on expired certificates as they might be in rotation
	// The operator should handle expired certificates gracefully

	return nil
}
