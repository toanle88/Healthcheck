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
func AuthMiddleware(tenantID, clientID string) gin.HandlerFunc {
	// 1. Initialize the JWKS key function
	jwksURL := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID)
	
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

		// Validate Audience (aud should be your clientId)
		aud, _ := claims["aud"].(string)
		if aud != clientID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid audience"})
			return
		}

		// Validate Issuer (iss)
		iss, _ := claims["iss"].(string)
		expectedIss := fmt.Sprintf("https://sts.windows.net/%s/", tenantID)
		if !strings.HasPrefix(iss, expectedIss) && iss != fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid issuer"})
			return
		}

		// Token is valid!
		c.Next()
	}
}
