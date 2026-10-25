import { expect, test } from '@playwright/test';

test('anonymous sandbox runs, inspects and exports an ephemeral result', async ({ page }) => {
  const requests: string[] = [];
  page.on('request', (request) => {
    if (request.url().includes('/api/')) requests.push(request.url());
  });
  await page.route('**/api/v1/demo/evaluations', async (route) => {
    expect(route.request().postDataJSON()).toEqual({
      profile: 'conditional',
      scenario: 'number_mismatch',
      strategy: 'staged',
    });
    await route.fulfill({
      json: {
        id: 'demo-fixture',
        policy_id: 'demo-conditional',
        policy_version: 1,
        profile: 'conditional',
        scenario: 'number_mismatch',
        strategy: 'staged',
        status: 'completed',
        public_demo: true,
        persisted: false,
        outcome: 'additional_verification_required',
        stopping_reason: 'Outcome established',
        supplier_cost_microrand: 1200,
        runtime_cost_microrand: 5,
        elapsed_ms: 12,
        plan: 'number_verification -> sim_swap -> device_swap',
        created_at: '2026-10-25T12:00:00Z',
        checks: [
          {
            capability: 'number_verification',
            status: 'available',
            value: true,
            source_time: '2026-10-25T12:00:00Z',
            started_at: '2026-10-25T12:00:00Z',
            elapsed_ms: 10,
            cost_microrand: 1200,
            compulsory: true,
          },
          ...['sim_swap', 'device_swap'].map((capability) => ({
            capability,
            status: 'skipped',
            source_time: '0001-01-01T00:00:00Z',
            started_at: '0001-01-01T00:00:00Z',
            elapsed_ms: 0,
            cost_microrand: 0,
            compulsory: false,
          })),
        ],
      },
    });
  });
  await page.goto('/demo');
  await expect(page.getByRole('heading', { name: /Follow a decision/ })).toBeVisible();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'Additional verification required' }),
  ).toBeVisible();
  await expect(page.getByText('Ephemeral result · not persisted')).toBeVisible();
  await page.getByText('Number verification', { exact: true }).click();
  await expect(
    page
      .locator('details')
      .filter({ hasText: 'Number verification' })
      .getByText('Source timestamp', { exact: true }),
  ).toBeVisible();
  const download = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Export JSON' }).click();
  expect((await download).suggestedFilename()).toBe('optitrace-demo-fixture.json');
  expect(requests).toHaveLength(1);
  expect(requests[0]).toContain('/demo/evaluations');
});

test('mobile sandbox shows capacity errors without navigating to login', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.route('**/api/v1/demo/evaluations', (route) =>
    route.fulfill({
      status: 429,
      json: {
        error: { code: 'demo_busy', message: 'The demo is busy. Try again in a few seconds.' },
      },
    }),
  );
  await page.goto('/demo');
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(page.getByRole('alert').filter({ hasText: 'The demo is busy' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Run verification', exact: true })).toBeEnabled();
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
  ).toBeTruthy();
  await expect(page).toHaveURL(/\/demo$/);
});
