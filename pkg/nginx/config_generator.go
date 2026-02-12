package nginx

import (
	"fmt"
	"strings"

	"github.com/jwks-operator/jwks-operator/pkg/config"
)

// ConfigGenerator generates nginx configuration for JWKS server
type ConfigGenerator struct {
	cacheMaxAge int
}

// NewConfigGenerator creates a new nginx config generator
func NewConfigGenerator(cacheMaxAge int) *ConfigGenerator {
	return &ConfigGenerator{
		cacheMaxAge: cacheMaxAge,
	}
}

// GenerateConfig generates nginx configuration for JWKS endpoint
func (g *ConfigGenerator) GenerateConfig(jwksConfigMapName string, endpoint string) (string, error) {
	if jwksConfigMapName == "" {
		return "", fmt.Errorf("JWKS ConfigMap name cannot be empty")
	}

	endpoint = NormalizeEndpoint(endpoint)
	if err := ValidateEndpoint(endpoint); err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	// Generate location block that serves jwks.json for all paths
	allPathsLocationBlock := g.GenerateAllPathsLocationBlock()

	// Generate server block with location block inside
	config := g.GenerateServerBlockWithLocations(config.DefaultNginxPort, allPathsLocationBlock, "")

	return config, nil
}

// GenerateServerBlock generates nginx server block
func (g *ConfigGenerator) GenerateServerBlock(port int) string {
	return fmt.Sprintf(`server {
    listen %d;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # Security headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
}`, port)
}

// GenerateServerBlockWithLocations generates nginx server block with location blocks inside
func (g *ConfigGenerator) GenerateServerBlockWithLocations(port int, rootLocationBlock, jwksLocationBlock string) string {
	config := fmt.Sprintf(`server {
    listen %d;
    server_name _;

    root /usr/share/nginx/html;

    # Security headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;

%s`, port, rootLocationBlock)

	if jwksLocationBlock != "" {
		config += fmt.Sprintf("\n\n%s", jwksLocationBlock)
	}

	config += "\n}"
	return config
}

// GenerateRootLocationBlock generates nginx location block for root path "/"
func (g *ConfigGenerator) GenerateRootLocationBlock(jwksPath string) string {
	return fmt.Sprintf(`    location = / {
        default_type application/json;
        alias %s;
        
        # CORS headers (if needed)
        add_header Access-Control-Allow-Origin "*" always;
        add_header Access-Control-Allow-Methods "GET, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type" always;
        
        # Cache control
        add_header Cache-Control "public, max-age=%d" always;
    }`, jwksPath, g.cacheMaxAge)
}

// GenerateAllPathsLocationBlock generates nginx location block that serves jwks.json for all paths
func (g *ConfigGenerator) GenerateAllPathsLocationBlock() string {
	// Always return /jwks.json for any path (including root /)
	// Use try_files to serve /jwks.json for all requests
	// This ensures root path / also returns the file
	return fmt.Sprintf(`    location / {
        default_type application/json;
        try_files /jwks.json =404;
        
        # CORS headers (if needed)
        add_header Access-Control-Allow-Origin "*" always;
        add_header Access-Control-Allow-Methods "GET, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type" always;
        
        # Cache control
        add_header Cache-Control "public, max-age=%d" always;
    }`, g.cacheMaxAge)
}

// GenerateLocationBlock generates nginx location block for JWKS endpoint
func (g *ConfigGenerator) GenerateLocationBlock(endpoint string, jwksPath string) string {
	// Ensure jwksPath starts with /
	if !strings.HasPrefix(jwksPath, "/") {
		jwksPath = "/" + jwksPath
	}

	return fmt.Sprintf(`    location %s {
        default_type application/json;
        alias %s;
        
        # CORS headers (if needed)
        add_header Access-Control-Allow-Origin "*" always;
        add_header Access-Control-Allow-Methods "GET, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type" always;
        
        # Cache control
        add_header Cache-Control "public, max-age=%d" always;
    }`, endpoint, jwksPath, g.cacheMaxAge)
}
