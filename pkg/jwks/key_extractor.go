package jwks

import (
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SHA-1 is required for certificate fingerprint per RFC standards
	"crypto/x509"
	"encoding/hex"
	"fmt"
)

// ExtractPublicKey extracts the public key from a certificate
func ExtractPublicKey(cert *x509.Certificate) (interface{}, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}

	return cert.PublicKey, nil
}

// ExtractRSAKey extracts an RSA public key from a certificate
func ExtractRSAKey(cert *x509.Certificate) (*rsa.PublicKey, error) {
	publicKey, err := ExtractPublicKey(cert)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate does not contain an RSA public key")
	}

	return rsaKey, nil
}

// GenerateKeyID generates a Key ID (kid) from a certificate
// Uses the first 16 characters of the SHA-1 fingerprint
func GenerateKeyID(cert *x509.Certificate) (string, error) {
	if cert == nil {
		return "", fmt.Errorf("certificate is nil")
	}

	// Generate SHA-1 fingerprint
	fingerprint := certFingerprint(cert)
	if len(fingerprint) < 16 {
		return "", fmt.Errorf("fingerprint too short")
	}

	// Use first 16 characters as kid
	return fingerprint[:16], nil
}

// certFingerprint generates a lowercase hex fingerprint from certificate
//
//nolint:gosec // SHA-1 is required for certificate fingerprint per RFC standards
func certFingerprint(cert *x509.Certificate) string {
	// Generate SHA-1 fingerprint
	hash := sha1.Sum(cert.Raw) //nolint:gosec // SHA-1 is required for certificate fingerprint
	return hex.EncodeToString(hash[:])
}
