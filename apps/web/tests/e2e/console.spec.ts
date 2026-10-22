import { test, expect, type Page } from '@playwright/test';
const policy = {
  id: 'policy-conditional',
  tenant_id: 't',
  name: 'Conditional verification',
  profile: 'conditional',
  version: 1,
  status: 'active',
  author_id: 'author',
  created_at: '2026-10-22T10:00:00Z',
};
async function setup(page: Page, roles = ['integration_developer', 'auditor']) {
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path.endsWith('/auth/me'))
      return route.fulfill({
        json: {
          user: { id: 'user', email: 'developer@optitrace.test' },
          tenants: [{ id: 't', name: 'South Africa demo', roles }],
          csrf_token: 'csrf',
        },
      });
    if (path.endsWith('/policies')) return route.fulfill({ json: { items: [policy] } });
    if (path.endsWith('/evaluations') && route.request().method() === 'POST')
      return route.fulfill({
        json: {
          id: 'trace-123',
          policy_id: policy.id,
          policy_version: 1,
          profile: 'conditional',
          scenario: 'sim_change',
          strategy: 'optitrace',
          outcome: 'additional_verification_required',
          stopping_reason: 'SIM change establishes the required outcome.',
          checks: [
            {
              capability: 'number_verification',
              status: 'available',
              value: false,
              compulsory: true,
              source_time: '2026-10-22T10:00:00Z',
              elapsed_ms: 12,
              cost_microrand: 1000,
            },
            {
              capability: 'sim_swap',
              status: 'available',
              value: true,
              compulsory: false,
              elapsed_ms: 15,
              cost_microrand: 2000,
            },
            {
              capability: 'device_swap',
              status: 'skipped',
              compulsory: false,
              elapsed_ms: 0,
              cost_microrand: 0,
            },
          ],
          supplier_cost_microrand: 3000,
          runtime_cost_microrand: 20,
          elapsed_ms: 28,
          created_at: '2026-10-22T10:00:00Z',
        },
      });
    return route.fulfill({ json: { items: [] } });
  });
}
test('runs a verification and inspects skipped evidence', async ({ page }) => {
  await setup(page);
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Verification playground' })).toBeVisible();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(page.getByText('SIM change establishes the required outcome.')).toBeVisible();
  await expect(page.getByText('Skipped', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('Synthetic data only.')).toBeVisible();
  await page.getByText('Device swap', { exact: true }).last().click();
  await expect(page.getByText('No source observation').last()).toBeVisible();
});
test('auditor cannot submit evaluations or administer membership', async ({ page }) => {
  await setup(page, ['auditor']);
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Run verification', exact: true })).toBeDisabled();
  await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Administration is restricted' })).toBeVisible();
});
test('unauthenticated visitors see the invited Google sign-in flow', async ({ page }) => {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({ status: 401, json: { error: { message: 'Authentication required' } } }),
  );
  await page.goto('/');
  await expect(page.getByRole('link', { name: 'Continue with Google' })).toHaveAttribute(
    'href',
    '/api/v1/auth/login',
  );
  await expect(page.getByRole('heading', { name: 'Your verification workspace' })).toBeVisible();
});
test('a queued benchmark polls until its measured report is available', async ({ page }) => {
  await setup(page);
  let launched = false;
  let polls = 0;
  const run = { id: 'job-123', seed: 42, count: 40, created_at: '2026-10-22T10:00:00Z' };
  await page.route('**/api/v1/tenants/t/benchmarks', async (route) => {
    if (route.request().method() === 'POST') {
      launched = true;
      return route.fulfill({ status: 202, json: { ...run, status: 'queued' } });
    }
    return route.fulfill({ json: { items: launched ? [{ ...run, status: 'queued' }] : [] } });
  });
  await page.route('**/api/v1/tenants/t/benchmarks/job-123', (route) =>
    route.fulfill({
      json: {
        ...run,
        status: ++polls > 1 ? 'completed' : 'running',
        results:
          polls > 1
            ? [
                {
                  strategy: 'optitrace',
                  profile: 'conditional',
                  measurement: 'measured_http',
                  agreement_rate: 1,
                  compulsory_violations: 0,
                  mean_cost_microrand: 4000,
                  p95_ms: 40,
                },
              ]
            : [],
      },
    }),
  );
  await page.goto('/');
  await page.getByRole('button', { name: 'Benchmarks', exact: true }).click();
  await page.getByRole('button', { name: 'Launch benchmark' }).click();
  await expect(page.getByText('Measured http', { exact: true })).toBeVisible({ timeout: 10000 });
  await expect(page.getByText('100.0%')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Export JSON' })).toHaveAttribute(
    'href',
    '/api/v1/tenants/t/benchmarks/job-123/export',
  );
});
test('the execution workspace remains usable on a narrow screen', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await setup(page);
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Run verification', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(page.getByText('SIM change establishes the required outcome.')).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  );
  await page.screenshot({ path: testInfo.outputPath('console-mobile.png'), fullPage: true });
});
