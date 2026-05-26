package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type mockTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

var testPrivKey *rsa.PrivateKey
var jwksJSON string

func init() {
	var err error
	testPrivKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	nBytes := testPrivKey.N.Bytes()
	nStr := base64.RawURLEncoding.EncodeToString(nBytes)

	eBytes := big.NewInt(int64(testPrivKey.E)).Bytes()
	eStr := base64.RawURLEncoding.EncodeToString(eBytes)

	jwksJSON = fmt.Sprintf(`{
		"keys": [
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "test-key-id",
				"alg": "RS256",
				"n": "%s",
				"e": "%s"
			}
		]
	}`, nStr, eStr)

	originalTransport := http.DefaultTransport
	http.DefaultTransport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Host, "ciamlogin.com") {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jwksJSON)),
					Header:     make(http.Header),
				}
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}
			return originalTransport.RoundTrip(req)
		},
	}
}

func createSignedToken(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-id"
	tokenString, err := token.SignedString(testPrivKey)
	if err != nil {
		panic(err)
	}
	return tokenString
}

func TestRequireRoleOrScope_NoClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 when claims do not exist, got %d", w.Code)
	}
}

func TestRequireRoleOrScope_InvalidClaimsType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", "invalid-type")
		c.Next()
	})
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for invalid claims type, got %d", w.Code)
	}
}

func TestRequireRoleOrScope_HasMatchingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", jwt.MapClaims{
			"roles": []interface{}{"User", "Admin"},
		})
		c.Next()
	})
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with matching role, got %d", w.Code)
	}
}

func TestRequireRoleOrScope_HasMatchingRoleString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", jwt.MapClaims{
			"roles": "Admin",
		})
		c.Next()
	})
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with matching single role string, got %d", w.Code)
	}
}

func TestRequireRoleOrScope_HasMatchingScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", jwt.MapClaims{
			"scp": "read write",
		})
		c.Next()
	})
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with matching scope, got %d", w.Code)
	}
}

func TestRequireRoleOrScope_InsufficientPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", jwt.MapClaims{
			"roles": []interface{}{"User"},
			"scp":   "read",
		})
		c.Next()
	})
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for insufficient permissions, got %d", w.Code)
	}
}

func TestAuthMiddleware_MockTokenBypass(t *testing.T) {
	t.Setenv("ALLOW_MOCK_AUTH", "true")
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "local"))
	r.GET("/test", RequireRoleOrScope([]string{"Healthcheck.Admin"}, nil), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer mocked-e2e-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 using mocked-e2e-token in local dev, got %d", w.Code)
	}
}

func TestAuthMiddleware_BypassQueryToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "local"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?token=mocked-e2e-token", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 using token query param (fallback disabled), got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "local"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing token, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "local"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "BearerInvalid mocked-e2e-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid Bearer format, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "production"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	claims := jwt.MapClaims{
		"tid": "dummy-tenant",
		"aud": "dummy-client",
		"iss": "https://dummy-tenant.ciamlogin.com/dummy-tenant/v2.0",
	}
	tokenString := createSignedToken(claims)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_TenantIDMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "production"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	claims := jwt.MapClaims{
		"tid": "wrong-tenant",
		"aud": "dummy-client",
		"iss": "https://dummy-tenant.ciamlogin.com/dummy-tenant/v2.0",
	}
	tokenString := createSignedToken(claims)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for tenant ID mismatch, got %d", w.Code)
	}
}

func TestAuthMiddleware_AudienceMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "production"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	claims := jwt.MapClaims{
		"tid": "dummy-tenant",
		"aud": "wrong-client",
		"iss": "https://dummy-tenant.ciamlogin.com/dummy-tenant/v2.0",
	}
	tokenString := createSignedToken(claims)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for audience mismatch, got %d", w.Code)
	}
}

func TestAuthMiddleware_IssuerMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(AuthMiddleware("dummy-tenant", "dummy-client", "production"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	claims := jwt.MapClaims{
		"tid": "dummy-tenant",
		"aud": "dummy-client",
		"iss": "https://wrong-tenant.ciamlogin.com/wrong-tenant/v2.0",
	}
	tokenString := createSignedToken(claims)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for issuer mismatch, got %d", w.Code)
	}
}

func TestMockAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(MockAuthMiddleware())
	r.GET("/test", RequireRoleOrScope([]string{"Healthcheck.Admin"}, nil), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 using MockAuthMiddleware, got %d", w.Code)
	}
}
