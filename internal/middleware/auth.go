package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates the Entra ID JWT token
func extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) == 2 && strings.ToLower(bearerToken[0]) == "bearer" {
			return bearerToken[1], nil
		}
		return "", fmt.Errorf("Invalid authorization header format")
	}

	if c.Request.URL.Path == "/api/status/stream" {
		tokenString := c.Query("token")
		if tokenString != "" {
			values := c.Request.URL.Query()
			values.Del("token")
			c.Request.URL.RawQuery = values.Encode()

			c.Request.RequestURI = c.Request.URL.Path
			if c.Request.URL.RawQuery != "" {
				c.Request.RequestURI += "?" + c.Request.URL.RawQuery
			}
			return tokenString, nil
		}
	}

	return "", fmt.Errorf("Authorization token is required")
}

func isMockTokenAllowed(tokenString, environment string) bool {
	return tokenString == "mocked-e2e-token" && environment == "local" && os.Getenv("ALLOW_MOCK_AUTH") == "true"
}

func validateClaims(claims jwt.MapClaims, tenantID, clientID string) error {
	tid, ok := claims["tid"].(string)
	if !ok || tid != tenantID {
		return fmt.Errorf("unauthorized tenant")
	}

	aud, _ := claims["aud"].(string)
	if aud != clientID {
		return fmt.Errorf("Invalid audience")
	}

	iss, _ := claims["iss"].(string)
	expectedIss := fmt.Sprintf("https://%s.ciamlogin.com/%s/v2.0", tenantID, tenantID)
	if iss != expectedIss {
		return fmt.Errorf("Invalid issuer")
	}

	return nil
}

// AuthMiddleware validates the Entra ID JWT token
func AuthMiddleware(tenantID, clientID, environment string) gin.HandlerFunc {
	// 1. Initialize the JWKS key function for CIAM
	jwksURL := fmt.Sprintf("https://%s.ciamlogin.com/%s/discovery/v2.0/keys", tenantID, tenantID)

	// Create the keyfunc strategy
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		panic(fmt.Sprintf("failed to create keyfunc: %v", err))
	}

	return func(c *gin.Context) {
		tokenString, err := extractToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if isMockTokenAllowed(tokenString, environment) {
			mockClaims := jwt.MapClaims{
				"roles": []interface{}{"Healthcheck.Admin"},
				"scp":   "Healthcheck.Write Healthcheck.Read",
			}
			c.Set("claims", mockClaims)
			c.Next()
			return
		}

		// 3. Parse and validate the token
		token, err := jwt.Parse(tokenString, k.Keyfunc)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid token: %v", err)})
			return
		}

		// 4. Validate claims (Audience and Issuer)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		if err := validateClaims(claims, tenantID, clientID); err != nil {
			if err.Error() == "unauthorized tenant" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			}
			return
		}

		// Token is valid!
		c.Set("claims", claims)
		c.Next()
	}
}

// RequireRoleOrScope checks if the authenticated user has at least one of the specified roles or scopes.
func RequireRoleOrScope(allowedRoles, allowedScopes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unauthenticated request"})
			return
		}

		claims, ok := claimsVal.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid claims structure"})
			return
		}

		hasRole := hasAnyRole(claims, allowedRoles)
		hasScope := hasAnyScope(claims, allowedScopes)

		if (len(allowedRoles) > 0 || len(allowedScopes) > 0) && !hasRole && !hasScope {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}

func hasAnyRole(claims jwt.MapClaims, allowedRoles []string) bool {
	if len(allowedRoles) == 0 {
		return false
	}
	rolesClaim, ok := claims["roles"]
	if !ok {
		return false
	}

	if rolesList, ok := rolesClaim.([]interface{}); ok {
		for _, r := range rolesList {
			if matchRole(r, allowedRoles) {
				return true
			}
		}
		return false
	}

	return matchRole(rolesClaim, allowedRoles)
}

func matchRole(role interface{}, allowedRoles []string) bool {
	rStr, ok := role.(string)
	if !ok {
		return false
	}
	for _, allowed := range allowedRoles {
		if rStr == allowed {
			return true
		}
	}
	return false
}

func hasAnyScope(claims jwt.MapClaims, allowedScopes []string) bool {
	if len(allowedScopes) == 0 {
		return false
	}
	scpClaim, ok := claims["scp"].(string)
	if !ok {
		return false
	}
	for _, s := range strings.Fields(scpClaim) {
		if matchScope(s, allowedScopes) {
			return true
		}
	}
	return false
}

func matchScope(scope string, allowedScopes []string) bool {
	for _, allowed := range allowedScopes {
		if scope == allowed {
			return true
		}
	}
	return false
}

// MockAuthMiddleware is used in local dev to bypass authentication by providing mock admin claims.
func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		mockClaims := jwt.MapClaims{
			"roles": []interface{}{"Healthcheck.Admin"},
			"scp":   "Healthcheck.Write Healthcheck.Read",
		}
		c.Set("claims", mockClaims)
		c.Next()
	}
}
