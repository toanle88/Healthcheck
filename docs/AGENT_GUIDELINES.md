# Healthcheck Developer & Agent Guidelines

This document establishes the styling standards, code structure, and best practices for developers and AI coding agents working on the **Healthcheck** repository. Adhering to these standards ensures code consistency, security, and maintainability across the backend, frontend, and infrastructure layers.

---

## 📂 Repository Code Structure

The repository follows a clean, modular structure. Any new code should fit into this layout:

```
.
├── .agents/            # Agent custom instructions and skill sets
├── cmd/                # Entry points for binaries
│   ├── api/            # Go REST API server
│   └── worker/         # Go background health check runner
├── docs/               # Project documentation and learning paths
├── internal/           # Private Go packages (not importable by other apps)
│   ├── config/         # Environment and Key Vault configuration
│   ├── handler/        # Gin HTTP handlers
│   ├── middleware/     # Gin middleware (CORS, Auth, etc.)
│   ├── monitor/        # OpenTelemetry instrumentation
│   └── store/          # Database access layer (PostgreSQL + pgx)
├── web/                # React frontend (Vite + TypeScript)
│   ├── src/
│   │   ├── components/ # React components (layout, dashboard, common)
│   │   ├── hooks/      # Custom React hooks (e.g., useAuth)
│   │   ├── pages/      # View pages (e.g., DashboardPage, LoginPage)
│   │   ├── services/   # API communication clients (Axios)
│   │   └── types/      # TypeScript interfaces and types
│   └── e2e/            # Playwright end-to-end tests
└── infra/              # Terraform Infrastructure-as-Code
    ├── envs/dev/       # Root development environment configuration
    └── modules/        # Reusable infrastructure modules
```

---

## 🐹 Go Backend Standards

### 1. Styling & Naming Conventions
- **Formatting**: All Go files must be formatted with `go fmt ./...`.
- **Naming**:
  - Exported functions, structs, and variables must use `PascalCase` and have brief doc comments.
  - Internal variables, functions, and parameters must use `camelCase`.
  - Use short, descriptive names for local variables (e.g., `ctx` for `context.Context`, `err` for `error`, `c` for `*gin.Context`).
  - Acronyms should be fully capitalized (e.g., `API`, `URL`, `ID`, `SLA`, `DB`).

### 2. HTTP Handlers (Gin)
- Handlers should be methods on a pointer-to-struct receiver (e.g., `func (h *Handler) Status(c *gin.Context)`).
- **Dependency Injection**: Handlers must not access the database or environment directly. They should use interfaces (e.g., `Storer`) passed to their constructors.
- **Context Pass-through**: Always use `c.Request.Context()` when calling store or config methods to respect context cancellation:
  ```go
  ctx := c.Request.Context()
  checks, err := h.store.GetLatestChecks(ctx)
  ```
- **HTTP Status Codes**: Use constant variables from the `net/http` package (e.g., `http.StatusOK`, `http.StatusInternalServerError`) instead of magic numbers (e.g., `200`, `500`).
- **Response Format**: Handlers should consistently return JSON payloads using `gin.H` or structured structs.

### 3. Database Layer (pgx/v5)
- Store all database interaction functions within `internal/store`.
- Use parameterized queries to protect against SQL injection.
- Prefer explicit type structures for scan operations.
- Avoid exposing database-specific errors directly to HTTP clients. Wrap or transform them to user-friendly messages.

### 4. Logging (`slog`)
- Use Go's structured logging library `log/slog`.
- Do not use print statements (`println`, `fmt.Println`) or standard `log` in production-bound code.
- Output formats:
  - **Local Development**: Human-readable text handler.
  - **Production/Docker**: JSON handler for ease of parsing by Azure Log Analytics / Application Insights.

### 5. Error Handling
- Never ignore errors with `_`. Handle them explicitly.
- Wrap errors with additional context when passing them up the stack using `fmt.Errorf("context: %w", err)`.
- In handlers, log the raw error and return a sanitized response to the user to prevent leakage of internal architecture details.

### 6. API Documentation (Scalar)
- API documentation is served dynamically at `/docs` using **Scalar**.
- The underlying OpenAPI 3.1.0 specification is stored in `docs/openapi.json`.
- The validation scripts (`check.sh` and `check.ps1`) automatically synchronize this file to `internal/handler/openapi.json` for binary embedding and serving at `/openapi.json`.
- Developers and agents modifying API routes or payloads should update `docs/openapi.json` and run the validation script to automatically synchronize and test the updates.

---

## ⚛️ Frontend Standards (React + TypeScript + Tailwind)

### 1. CSS & Design System
- **Aesthetic**: The application uses a **dark-theme premium aesthetic** (slate backgrounds like `bg-slate-900`, `bg-slate-950`).
- **Utility First**: Style components strictly using **Tailwind CSS 4** utility classes. Avoid writing custom CSS rules in files unless Tailwind cannot handle the requirement.
- **Visual Enhancements**:
  - Use glassmorphism (`bg-slate-900/50 backdrop-blur-md border-slate-800`).
  - Add smooth state changes (`transition-all duration-300`).
  - Implement active hover states (e.g., `hover:border-slate-700`, `hover:shadow-indigo-500/5`).
  - Utilize vibrant theme accents: Indigo/Violet for main components, Emerald for success/up states, Amber for warning/pending, Rose/Red for error/down states.

### 2. React Components
- All components should be defined as typed functional components (`React.FC` or regular function components with typed props).
- Components must be highly focused and single-purpose.
- Prefer relative path imports within the `web/src` folder.

### 3. Data Fetching & State
- Use **TanStack Query (React Query)** for remote server state. Do not use raw `useEffect` blocks to fetch API data.
- Centralize axios client configuration inside `web/src/services/api.ts` or similar endpoints.
- Avoid global client-side state unless strictly necessary. Rely on React Query's caching mechanisms.

### 4. TypeScript Strictness
- Never use the `any` type. Define interfaces or types for all objects, components, and API responses in `web/src/types`.
- Use TypeScript's strict mode properties (e.g., optional chaining `?.`, nullish coalescing `??`).

---

## ☁️ Infrastructure Standards (Terraform)

- **Modularity**: Infrastructure must be partitioned into functional modules located in `infra/modules/`.
- **Zero-Secret Architecture**:
  - Never commit credentials, passwords, or connection strings.
  - Utilize **Azure Managed Identities** (User-Assigned Managed Identity) for secure resource access (Key Vault, PostgreSQL).
  - Inject database configuration using Azure Key Vault secret URIs rather than plaintext values.
- **Network Isolation**: PostgreSQL databases and compute subnets must be isolated using VNets, private subnets, and private DNS zones.
- **Formatting**: Always execute `terraform fmt -recursive` on the `infra/` folder.

---

## 🧪 Testing & Verification Standards

All code modifications must be verified through the appropriate automated checks before committing.

- **Backend Tests**:
  - Unit tests belong in the same directory as the target code, named `*_test.go`.
  - Mock external interfaces (like `Storer`) to keep tests fast, hermetic, and independent of external resources.
  - Executed via: `go test -v -race ./...`
- **Frontend Tests**:
  - Unit tests belong in the same directory as the target component, named `*.test.tsx`.
  - Use **Vitest** along with **Mock Service Worker (MSW)** to mock network responses.
- **End-to-End Tests**:
  - Playwright integration tests reside in `web/e2e/`.
- **Validation Script**:
  - Execute `bash check.sh` (or `powershell ./check.ps1` on Windows) to format, compile, lint, and run the entire verification suite.

---

## 🤖 Guidelines for AI Coding Agents

When implementing changes in this repository, agents must adhere to the following operational guardrails:

1. **Maintain Documentation Integrity**: Preserve all existing comments, docstrings, and documentation unless specifically requested to update them.
2. **No Placeholders**: Never leave `TODO` items or empty function blocks in production code. Generate complete, fully-functional implementations.
3. **Run Pre-Commit Checks**: Always execute `check.sh` locally to verify there are no compilation, linting, or formatting errors.
4. **Follow the Sandbox Policy**: For tasks requiring network access (like package installations or git operations), explicitly set `BypassSandbox: true` and request user approval. Keep local compilations and test executions sandboxed.
5. **Leverage Code Review Skills**: If unsure about the design patterns or style, refer to the available custom skills:
   - `code-review` to double-check style alignment.
   - `bug-review` to resolve failing tests or compilation errors.
