package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

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
		// 2. Extract the token from the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := bearerToken[1]

		// 2b. E2E / Development mock token bypass (MFA / redirect automation support)
		isLocalDev := environment == "local"
		if tokenString == "mocked-e2e-token" && isLocalDev {
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

		// Verify the tenant ID (tid) to ensure it's from our specific tenant
		if tid, ok := claims["tid"].(string); !ok || tid != tenantID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unauthorized tenant"})
			return
		}

		// Validate Audience (aud should be your clientId)
		aud, _ := claims["aud"].(string)
		if aud != clientID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid audience"})
			return
		}

		// Validate Issuer (iss for CIAM)
		iss, _ := claims["iss"].(string)
		expectedIss := fmt.Sprintf("https://%s.ciamlogin.com/%s/v2.0", tenantID, tenantID)
		if iss != expectedIss {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid issuer"})
			return
		}

		// Token is valid!
		c.Set("claims", claims)
		c.Next()
	}
}

// RequireRoleOrScope checks if the authenticated user has at least one of the specified roles or scopes.
func RequireRoleOrScope(allowedRoles []string, allowedScopes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("claims")
		if !exists {
			// Auth middleware not run (e.g. disabled in local dev). Allow.
			c.Next()
			return
		}

		claims, ok := claimsVal.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid claims structure"})
			return
		}

		hasRole := false
		if len(allowedRoles) > 0 {
			if rolesClaim, ok := claims["roles"]; ok {
				if rolesList, ok := rolesClaim.([]interface{}); ok {
					for _, r := range rolesList {
						if rStr, ok := r.(string); ok {
							for _, allowed := range allowedRoles {
								if rStr == allowed {
									hasRole = true
									break
								}
							}
						}
						if hasRole {
							break
						}
					}
				} else if rStr, ok := rolesClaim.(string); ok {
					for _, allowed := range allowedRoles {
						if rStr == allowed {
							hasRole = true
							break
						}
					}
				}
			}
		}

		hasScope := false
		if len(allowedScopes) > 0 {
			if scpClaim, ok := claims["scp"].(string); ok {
				scopes := strings.Fields(scpClaim)
				for _, s := range scopes {
					for _, allowed := range allowedScopes {
						if s == allowed {
							hasScope = true
							break
						}
					}
					if hasScope {
						break
					}
				}
			}
		}

		if (len(allowedRoles) > 0 || len(allowedScopes) > 0) && !hasRole && !hasScope {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}
