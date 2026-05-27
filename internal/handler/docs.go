package handler

import (
	// embed is required to use the go:embed directive for openapi.json
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.json
var openAPISpec []byte

// OpenAPISpec serves the raw/dynamically generated JSON OpenAPI specification.
// It dynamically injects Entra ID configuration (authorize URL, token URL, client ID) if configured.
//
// OpenAPISpec godoc
// @Summary Get raw OpenAPI specification
// @Description Serves the raw JSON OpenAPI specification.
// @Tags Docs
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /openapi.json [get]
func (h *Handler) OpenAPISpec(c *gin.Context) {
	tenantID := os.Getenv("ENTRA_TENANT_ID")
	clientID := os.Getenv("ENTRA_CLIENT_ID")
	tenantDomain := os.Getenv("ENTRA_TENANT_DOMAIN")
	if tenantDomain == "" {
		tenantDomain = "toanlesandbox.ciamlogin.com"
	}

	if tenantID == "" || clientID == "" {
		c.Data(http.StatusOK, "application/json", openAPISpec)
		return
	}

	// 1. Replace authorizationUrl and tokenUrl
	authURL := fmt.Sprintf("https://%s/%s/oauth2/v2.0/authorize", tenantDomain, tenantID)
	tokenURL := fmt.Sprintf("https://%s/%s/oauth2/v2.0/token", tenantDomain, tenantID)

	specStr := string(openAPISpec)
	specStr = strings.Replace(specStr, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", authURL, 1)
	specStr = strings.Replace(specStr, "https://login.microsoftonline.com/common/oauth2/v2.0/token", tokenURL, 1)

	// 2. Replace scopes
	apiScope := fmt.Sprintf("api://%s/access_as_user", clientID)
	scopesStr := fmt.Sprintf(`"scopes": {
            "%s": "Access Healthcheck API as user",
            "openid": "Sign you in",
            "profile": "Read your profile",
            "email": "Read your email address",
            "offline_access": "Maintain access to data"
          }`, apiScope)
	specStr = strings.Replace(specStr, `"scopes": {}`, scopesStr, 1)

	c.Data(http.StatusOK, "application/json", []byte(specStr))
}

// Docs renders the interactive Scalar API documentation.
// It serves an HTML template embedding the Scalar API reference interface.
//
// Docs godoc
// @Summary Render interactive API documentation
// @Description Serves the Scalar HTML interface for interactive API exploration.
// @Tags Docs
// @Produce html
// @Success 200 {string} string "HTML page"
// @Router /docs [get]
func (h *Handler) Docs(c *gin.Context) {
	clientID := os.Getenv("ENTRA_CLIENT_ID")

	// Set BearerAuth as the default preferred security scheme for convenience
	configJSON := `{"theme": "purple", "showSidebar": true, "layout": "modern", "authentication": {"preferredSecurityScheme": "BearerAuth"}}`
	if clientID != "" {
		configJSON = fmt.Sprintf(`{"theme": "purple", "showSidebar": true, "layout": "modern", "authentication": {"preferredSecurityScheme": "BearerAuth", "oAuth2": {"clientId": "%s"}}}`, clientID)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
  <head>
    <title>Healthcheck API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      body {
        margin: 0;
      }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/openapi.json"
      data-configuration='%s'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`, configJSON)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
