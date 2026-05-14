# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: dashboard.spec.ts >> has title and displays status
- Location: e2e/dashboard.spec.ts:3:1

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('h1')
Expected substring: "Healthcheck"
Received string:    "Welcome Back"
Timeout: 5000ms

Call log:
  - Expect "toContainText" with timeout 5000ms
  - waiting for locator('h1')
    13 × locator resolved to <h1 class="text-3xl font-bold text-white mb-3">Welcome Back</h1>
       - unexpected value "Welcome Back"

```

```yaml
- heading "Welcome Back" [level=1]
```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | 
  3  | test('has title and displays status', async ({ page }) => {
  4  |   // Navigate to the local dev server
  5  |   await page.goto('http://localhost:5173');
  6  | 
  7  |   // Check the title
  8  |   await expect(page).toHaveTitle(/Healthcheck Dashboard/);
  9  | 
  10 |   // Check for the main header
  11 |   const header = page.locator('h1');
> 12 |   await expect(header).toContainText('Healthcheck');
     |                        ^ Error: expect(locator).toContainText(expected) failed
  13 | 
  14 |   // Since E2E tests run against the real backend, 
  15 |   // we check if the operational status indicator is visible.
  16 |   // Note: This requires the backend and worker to be running!
  17 |   const statusIndicator = page.locator('p', { hasText: 'System' });
  18 |   await expect(statusIndicator).toBeVisible();
  19 | });
  20 | 
```