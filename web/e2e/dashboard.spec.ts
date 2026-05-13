import { test, expect } from '@playwright/test';

test('has title and displays status', async ({ page }) => {
  // Navigate to the local dev server
  await page.goto('http://localhost:5173');

  // Check the title
  await expect(page).toHaveTitle(/Healthcheck Dashboard/);

  // Check for the main header
  const header = page.locator('h1');
  await expect(header).toContainText('Healthcheck');

  // Since E2E tests run against the real backend, 
  // we check if the operational status indicator is visible.
  // Note: This requires the backend and worker to be running!
  const statusIndicator = page.locator('p', { hasText: 'System' });
  await expect(statusIndicator).toBeVisible();
});
