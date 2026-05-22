package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequireRoleOrScope_NoClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequireRoleOrScope([]string{"Admin"}, []string{"write"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when claims do not exist (auth bypassed), got %d", w.Code)
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
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Set up AuthMiddleware with environment "local" to trigger mock bypass
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
