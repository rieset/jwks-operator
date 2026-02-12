package verification

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	corev1 "k8s.io/api/core/v1"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// Verifier verifies JWKS served by nginx
type Verifier struct {
	httpClient *http.Client
	config     *config.VerificationConfig
}

// NewVerifier creates a new JWKS verifier
func NewVerifier(verificationConfig *config.VerificationConfig) *Verifier {
	timeout := config.DefaultVerificationTimeout
	if verificationConfig != nil && verificationConfig.Timeout > 0 {
		timeout = verificationConfig.Timeout
	}

	return &Verifier{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config: verificationConfig,
	}
}

// VerifyJWKSFromNginx verifies that JWKS served by nginx can verify JWT tokens signed with the certificate's private key
func (v *Verifier) VerifyJWKSFromNginx(
	ctx context.Context,
	namespace string,
	serviceName string,
	secret *corev1.Secret,
) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}

	// Step 1: Get JWKS from nginx Service
	jwksURL := fmt.Sprintf("http://%s.%s.svc.cluster.local/jwks.json", serviceName, namespace)
	jwksData, err := v.fetchJWKS(ctx, jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS from nginx: %w", err)
	}

	// Step 2: Extract public key from JWKS
	publicKey, kid, err := v.extractPublicKeyFromJWKS(jwksData)
	if err != nil {
		return fmt.Errorf("failed to extract public key from JWKS: %w", err)
	}

	// Step 3: Extract private key from certificate secret
	privateKey, err := v.extractPrivateKeyFromSecret(secret)
	if err != nil {
		return fmt.Errorf("failed to extract private key from secret: %w", err)
	}

	// Step 4: Create a test JWT token signed with private key
	testToken, err := v.createTestJWT(privateKey, kid)
	if err != nil {
		return fmt.Errorf("failed to create test JWT: %w", err)
	}

	// Step 5: Verify the token using public key from JWKS
	if err := v.verifyJWT(testToken, publicKey); err != nil {
		return fmt.Errorf("failed to verify JWT with public key from JWKS: %w", err)
	}

	return nil
}

// fetchJWKS fetches JWKS from nginx Service
func (v *Verifier) fetchJWKS(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// JWKS represents the JWKS structure
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JWK
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"` // RSA modulus (base64url)
	E   string `json:"e"` // RSA exponent (base64url)
}

// extractPublicKeyFromJWKS extracts RSA public key from JWKS
func (v *Verifier) extractPublicKeyFromJWKS(jwksData []byte) (*rsa.PublicKey, string, error) {
	var jwks JWKS
	if err := json.Unmarshal(jwksData, &jwks); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal JWKS: %w", err)
	}

	if len(jwks.Keys) == 0 {
		return nil, "", fmt.Errorf("JWKS contains no keys")
	}

	// Use the first key
	jwk := jwks.Keys[0]
	if jwk.Kty != "RSA" {
		return nil, "", fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}

	// Decode base64url encoded modulus
	n, err := decodeBase64URL(jwk.N)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url encoded exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent bytes to int
	var expInt int
	for _, b := range eBytes {
		expInt = expInt<<8 | int(b)
	}

	publicKey := &rsa.PublicKey{
		N: n,
		E: expInt,
	}

	return publicKey, jwk.Kid, nil
}

// extractPrivateKeyFromSecret extracts RSA private key from Kubernetes Secret
func (v *Verifier) extractPrivateKeyFromSecret(secret *corev1.Secret) (*rsa.PrivateKey, error) {
	keyData, ok := secret.Data[config.SecretKeyTLSKey]
	if !ok {
		return nil, fmt.Errorf("%s not found in secret", config.SecretKeyTLSKey)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}

		return rsaKey, nil
	}

	return privateKey, nil
}

// createTestJWT creates a test JWT token signed with the private key
func (v *Verifier) createTestJWT(privateKey *rsa.PrivateKey, kid string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "jwks-operator-verification",
		"sub": "test",
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"kid": kid,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// verifyJWT verifies a JWT token using the public key
func (v *Verifier) verifyJWT(tokenString string, publicKey *rsa.PublicKey) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("token is not valid")
	}

	return nil
}

// decodeBase64URL decodes base64url encoded string
func decodeBase64URL(s string) (*big.Int, error) {
	// Base64URL decoding
	decoded, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64url: %w", err)
	}

	// Convert to big.Int
	result := new(big.Int).SetBytes(decoded)
	return result, nil
}
