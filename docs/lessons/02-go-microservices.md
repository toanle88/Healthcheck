# Lesson 02: Go Microservices 🐹

Our Go application is designed to be **Cloud-Native**. This means it doesn't just run *on* Azure; it *integrates* with Azure services.

## 1. The Entry Points (`cmd/`)

We have two separate binaries built from the same codebase:
- **`cmd/api/main.go`**: The long-running web server (using the Gin framework).
- **`cmd/worker/main.go`**: The short-lived task runner.

They share the same business logic in the `internal/` folder, which keeps our code DRY (Don't Repeat Yourself).

## 2. Passwordless Database Connections (`internal/store`)

This is the "Magic" part of the project. Look at `internal/store/postgres.go`.

```go
// The secret sauce: Azure AD Token Authentication
cfg.BeforeConnect = func(ctx context.Context, pgc *pgx.ConnConfig) error {
    token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
        Scopes: []string{"https://ossrdbms-aad.database.windows.net/.default"},
    })
    if err != nil {
        return fmt.Errorf("failed to get azure ad token: %w", err)
    }
    pgc.Password = token.Token // We use the TOKEN as the password!
    return nil
}
```

**Why we do this:**
Normally, you'd put a password in a `.env` file. If that file is leaked, your database is gone. By using **Managed Identity tokens**, the token expires every hour, and it can *only* be requested by a container running inside your specific Azure environment.

## 3. Observability (`internal/monitor`)

We use **OpenTelemetry (OTel)**. Instead of just printing "Error happened," our code sends structured data to **Azure Application Insights**.

- **Metrics**: We track how many requests are happening and how long they take (P95 latency).
- **Traces**: We can follow a single request as it goes from the Web frontend -> API -> Database.

## 4. Secure Middleware (`internal/middleware`)

The API doesn't trust anyone. Every request coming from the Frontend must have a valid **Entra ID JWT Token**.

In `middleware/auth.go`, we:
1.  Extract the `Authorization: Bearer <token>` header.
2.  Validate it against your Azure CIAM tenant.
3.  Only if the token is valid does the code allow the request to reach the database.

## 5. Embedded API Documentation (Scalar) 📖🔍

To make developer onboarding seamless, the API embeds its own interactive documentation using **Scalar** and the standard OpenAPI 3.1.0 specification.

- **Single-Binary Distribution (`go:embed`)**: The OpenAPI JSON schema (`openapi.json`) is embedded directly into the Go binary at build time. This ensures that the documentation is always packaged with the application and never gets out of sync or lost.
- **Dynamic Configuration**: The server dynamically rewrites the OpenAPI authorization and token URLs to point to the active Microsoft Entra ID tenant (from the `ENTRA_TENANT_ID` and `ENTRA_CLIENT_ID` environment variables) when served.
- **Dual Authentication Modes**: Developers can authenticate via the standard Entra ID redirect login flow (pre-filled with the active Client ID) or manually paste a Bearer JWT token directly into the Scalar console.

---

### Key Takeaway
The Go code is "Identity-Aware." It knows who it is (Managed Identity) and it knows who the user is (Entra ID Token). This creates a solid chain of trust from the browser all the way to the database disk.

Next: **Lesson 03 — Infrastructure as Code (Terraform)**.
