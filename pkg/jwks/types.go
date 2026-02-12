package jwks

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	// Key type (e.g., "RSA")
	Kty string `json:"kty"`

	// Public key use (e.g., "sig")
	Use string `json:"use"`

	// Key operations (e.g., ["verify"])
	KeyOps []string `json:"key_ops,omitempty"`

	// Algorithm (e.g., "RS512")
	Alg string `json:"alg"`

	// Key ID
	Kid string `json:"kid"`

	// RSA modulus (base64url encoded)
	N string `json:"n,omitempty"`

	// RSA exponent (base64url encoded)
	E string `json:"e,omitempty"`

	// X.509 certificate chain (base64 encoded)
	X5c []string `json:"x5c,omitempty"`

	// X.509 certificate SHA-1 thumbprint (base64url encoded)
	X5t string `json:"x5t,omitempty"`

	// X.509 certificate SHA-256 thumbprint (base64url encoded)
	X5tS256 string `json:"x5t#S256,omitempty"`
}
