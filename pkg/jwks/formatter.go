package jwks

import (
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SHA-1 is required for x5t thumbprint per RFC 7517
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// FormatJWK formats a public key as a JWK
func FormatJWK(key interface{}, kid string, cert *x509.Certificate) (*JWK, error) {
	switch k := key.(type) {
	case *rsa.PublicKey:
		return FormatRSAKey(k, kid, cert)
	default:
		return nil, fmt.Errorf("unsupported key type: %T", key)
	}
}

// FormatRSAKey formats an RSA public key as a JWK
func FormatRSAKey(key *rsa.PublicKey, kid string, cert *x509.Certificate) (*JWK, error) {
	if key == nil {
		return nil, fmt.Errorf("RSA key is nil")
	}

	// Encode modulus (n) - base64url encoding
	nBytes := key.N.Bytes()
	nBase64 := base64.RawURLEncoding.EncodeToString(nBytes)

	// Encode exponent (e) - base64url encoding
	eBytes := big.NewInt(int64(key.E)).Bytes()
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	jwk := &JWK{
		Kty:    "RSA",
		Use:    "sig",
		KeyOps: []string{"verify"},
		Alg:    "RS512",
		Kid:    kid,
		N:      nBase64,
		E:      eBase64,
	}

	// Add certificate chain if available
	if cert != nil {
		// Encode certificate as base64
		certDER := cert.Raw
		certBase64 := base64.StdEncoding.EncodeToString(certDER)
		jwk.X5c = []string{certBase64}

		// Calculate thumbprints
		jwk.X5t = calculateX5t(certDER)
		jwk.X5tS256 = calculateX5tS256(certDER)
	}

	return jwk, nil
}

// ToJSON converts JWKS to JSON
func ToJSON(jwks *JWKS) ([]byte, error) {
	if jwks == nil {
		return nil, fmt.Errorf("JWKS is nil")
	}

	return json.MarshalIndent(jwks, "", "  ")
}

// calculateX5t calculates SHA-1 thumbprint (x5t)
//
//nolint:gosec // SHA-1 is required for x5t thumbprint per RFC 7517
func calculateX5t(certDER []byte) string {
	hash := sha1.Sum(certDER) //nolint:gosec // SHA-1 is required for x5t thumbprint per RFC 7517
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// calculateX5tS256 calculates SHA-256 thumbprint (x5t#S256)
func calculateX5tS256(certDER []byte) string {
	hash := sha256.Sum256(certDER)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
