# Healthcheck Dashboard Client (React + TypeScript)

This is the front-end dashboard client for the Healthcheck project. It is built using **React 19**, **Vite**, **TypeScript**, and **Tailwind CSS v4**.

It communicates with the Go API backend to provide real-time endpoint monitoring status, history graphs, targets administration, and OAuth2/Entra authentication.

---

## 📦 Tech Stack

- **Framework**: React 19 + TypeScript + Vite
- **Styling**: Tailwind CSS v4
- **State & Data Fetching**: TanStack React Query v5 + Axios
- **Authentication**: `@azure/msal-react` & `@azure/msal-browser` (Entra External ID / CIAM)
- **Icons**: Lucide React
- **Unit Testing**: Vitest + React Testing Library + Mock Service Worker (MSW)
- **E2E Testing**: Playwright

---

## 🎨 Features

1. **Enterprise Authentication**: SSO integration with Microsoft Entra External ID (CIAM) using the Authorization Code Flow with PKCE.
2. **Real-time Status Feed**: Automated refresh loops querying target latencies and state checks every 10 seconds.
3. **Dynamic Targets Administration**: Add new URLs (validated) and delete existing targets instantly via standard REST API endpoints.
4. **SLA Badges**: 24h SLA calculation shown with color-coded safety margins.
5. **Visual Health Sparklines**: Interactive SVG sparklines rendering the last 30 pings of latency history, plus chronological status tick bars showing detailed timeline patterns.

---

## 📁 Project Structure

```
web/
├── e2e/                     # Playwright E2E test suite
├── public/                  # Static assets and runtime environment files
├── src/
│   ├── assets/              # Icons and images
│   ├── components/
│   │   ├── common/          # ErrorDisplay, LoadingSpinner
│   │   ├── dashboard/       # HealthCard, UptimeChart, TargetsHeader, TargetModal
│   │   └── layout/          # Header, Footer
│   ├── config/              # Runtime environment variables loader
│   ├── hooks/               # Custom React hooks (useAuth, useHealthQuery)
│   ├── lib/                 # Shared Axios client configuration
│   ├── pages/               # LoginPage, DashboardPage
│   ├── services/            # API client service layer (healthService)
│   ├── test/                # Test utilities, setups, and MSW mocks
│   ├── types/               # TypeScript interface schemas (Check, Target)
│   ├── App.tsx              # Application entrypoint with MSAL/Query providers
│   ├── authConfig.ts        # MSAL configuration (client IDs, scopes)
│   └── main.tsx             # DOM mounting
├── Dockerfile.web           # Nginx-based multi-stage container
└── package.json
```

---

## 🚀 Running Locally

Ensure you have [Node.js](https://nodejs.org/) and `pnpm` installed.

### 1. Install dependencies
```bash
pnpm install
```

### 2. Configure Environment variables
The application reads settings at runtime from `public/env.js` (locally) or injected in Docker.
Copy `.env.example` to `.env` or check `web/.env.example`.
Standard local API port is `http://localhost:8080`.

### 3. Run development server (HMR enabled)
```bash
pnpm run dev
```
Open [http://localhost:5173](http://localhost:5173) in your browser.

### 4. Build for production
```bash
pnpm run build
```
This generates the optimized bundle in the `dist/` directory, ready to be served by Nginx.

---

## 🧪 Testing

The client includes unit and integration tests covering components, pages, custom hooks, and service layers.

### Run Unit Tests (Vitest)
```bash
pnpm run test:unit
```

All API requests in tests are mocked via **Mock Service Worker (MSW)** defined in `src/test/mocks/handlers.ts` to ensure consistent, offline test coverage.
