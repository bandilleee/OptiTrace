import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
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
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Verification playground' })).toBeVisible();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(
    page.locator('.outcome-row').getByText('SIM change establishes the required outcome.'),
  ).toBeVisible();
  await expect(page.getByText('Skipped', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('Synthetic sandbox.', { exact: true })).toBeVisible();
  await page.locator('.trace-item').filter({ hasText: 'Device swap' }).locator('summary').click();
  await expect(page.getByText('No source observation').last()).toBeVisible();
});
test('auditor cannot submit evaluations or administer membership', async ({ page }) => {
  await setup(page, ['auditor']);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Workspace overview' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Playground', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Administration', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Audit history', exact: true })).toBeVisible();
});
test('unauthenticated visitors see the invited Google sign-in flow', async ({ page }) => {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({ status: 401, json: { error: { message: 'Authentication required' } } }),
  );
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await expect(page.getByRole('link', { name: 'Continue with Google' })).toHaveAttribute(
    'href',
    '/api/v1/auth/login',
  );
  await expect(page.getByRole('heading', { name: 'Your verification workspace' })).toBeVisible();
});
test('a queued benchmark polls until its measured report is available', async ({
  page,
}, testInfo) => {
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
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await page.getByRole('button', { name: 'Benchmarks', exact: true }).click();
  await page.getByRole('button', { name: 'Launch benchmark' }).click();
  await expect(page.getByText('Measured http', { exact: true })).toBeVisible({ timeout: 10000 });
  await expect(page.getByText('100.0%')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Export JSON' })).toHaveAttribute(
    'href',
    '/api/v1/tenants/t/benchmarks/job-123/export',
  );
  await expect(page.getByRole('dialog', { name: 'Comparison report' })).toBeVisible();
  const accessibility = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze();
  expect(accessibility.violations, JSON.stringify(accessibility.violations, null, 2)).toEqual([]);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Launch benchmark' })).toBeFocused();
  await page.locator('.run-row').click();
  await expect(page.getByRole('dialog', { name: 'Comparison report' })).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  );
  await page.screenshot({
    path: testInfo.outputPath('benchmark-popup-mobile.png'),
    fullPage: true,
  });
});
test('the execution workspace remains usable on a narrow screen', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await setup(page);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Run verification', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(
    page.locator('.outcome-row').getByText('SIM change establishes the required outcome.'),
  ).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  );
  await page.screenshot({ path: testInfo.outputPath('console-mobile.png'), fullPage: true });
});

test('theme and motion choices persist, and the assumptions match the engine', async ({ page }) => {
  await setup(page);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await page.getByRole('button', { name: 'Switch to light theme' }).click();
  await page.getByRole('button', { name: 'Pause motion' }).click();
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await expect(page.getByRole('button', { name: 'Play motion' })).toBeVisible();
  await expect(page.locator('.sidebar .logo img').first()).toHaveAttribute(
    'src',
    '/brand/static.svg',
  );
  await page.getByRole('button', { name: 'View assumptions' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('60 seconds');
  await expect(dialog).toContainText('24 hours');
  await page.keyboard.press('Escape');
  await expect(dialog).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'View assumptions' })).toBeFocused();
});

test('mobile navigation closes after selection and tabs support the keyboard', async ({ page }) => {
  await setup(page);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await page.getByRole('button', { name: 'Open navigation' }).click();
  await page
    .getByRole('dialog')
    .getByRole('button', { name: 'Execution traces', exact: true })
    .click();
  await expect(page.getByRole('heading', { name: 'Execution traces', exact: true })).toBeVisible();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.getByRole('button', { name: 'Open navigation' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Playground', exact: true }).click();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await page.getByRole('tab', { name: 'Execution trace', exact: true }).focus();
  await page.keyboard.press('ArrowRight');
  await expect(page.getByRole('tab', { name: 'Run inputs' })).toHaveAttribute(
    'data-state',
    'active',
  );
  await expect(page.getByRole('tabpanel')).toContainText('60 seconds');
});

test('failed submissions show the server error without inventing a result', async ({ page }) => {
  await setup(page);
  await page.route('**/api/v1/tenants/t/evaluations', (route) =>
    route.request().method() === 'POST'
      ? route.fulfill({
          status: 503,
          json: { error: { message: 'Source execution is unavailable. Try again.' } },
        })
      : route.fulfill({ json: { items: [] } }),
  );
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(page.locator('.error[role=alert]')).toContainText('Source execution is unavailable');
  await expect(page.locator('.outcome-row')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Run verification', exact: true })).toBeEnabled();
});

test('workspace has no automated accessibility violations in either theme', async ({ page }) => {
  await setup(page);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: /Workspace overview|Your verification workspace/ }),
  ).toBeVisible();
  if (await page.getByRole('button', { name: 'Open verification playground', exact: true }).count())
    await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  await page.getByRole('button', { name: 'Run verification', exact: true }).click();
  await expect(page.locator('.outcome-row')).toBeVisible();
  for (const theme of ['dark', 'light']) {
    if (theme === 'light')
      await page.getByRole('button', { name: 'Switch to light theme' }).click();
    const result = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();
    expect(result.violations, JSON.stringify(result.violations, null, 2)).toEqual([]);
  }
});

test('policy authors see their own dashboard without evaluation or administration access', async ({
  page,
}) => {
  const requests: string[] = [];
  page.on('request', (request) => requests.push(new URL(request.url()).pathname));
  await setup(page, ['policy_author']);
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Workspace overview' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Policy author', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Benchmarks', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Administration', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: 'Write and submit policies' }).click();
  await page.getByLabel('Policy name', { exact: true }).fill('   ');
  await page.getByRole('button', { name: 'Create draft', exact: true }).click();
  await expect(page.getByText('Enter a name.', { exact: true })).toBeVisible();
  expect(requests.some((path) => path.endsWith('/evaluations'))).toBe(false);
});

test('dropdowns support keyboard selection and invalid benchmark values stay in the form', async ({
  page,
}) => {
  await setup(page);
  await page.goto('/');
  await page.getByRole('button', { name: 'Open verification playground', exact: true }).click();
  const strategy = page.getByRole('combobox', { name: 'Execution strategy', exact: true });
  await strategy.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('listbox')).toBeVisible();
  await page.keyboard.press('End');
  await page.keyboard.press('Enter');
  await expect(page.getByRole('listbox')).toHaveCount(0);
  await expect(strategy).toBeFocused();
  await page.getByRole('button', { name: 'Benchmarks', exact: true }).click();
  await page.getByLabel('Observation count').fill('1');
  await page.getByRole('button', { name: 'Launch benchmark' }).click();
  await expect(
    page.getByText('Enter a whole number between 2 and 1000.', { exact: true }),
  ).toBeVisible();
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('administrators validate invitations and retain failed role edits for retry', async ({
  page,
}) => {
  await setup(page, ['tenant_admin']);
  let attempts = 0;
  await page.route('**/api/v1/tenants/t/memberships', (route) =>
    route.fulfill({
      json: { items: [{ id: 'member', email: 'person@example.com', roles: ['auditor'] }] },
    }),
  );
  await page.route('**/api/v1/tenants/t/memberships/member', (route) => {
    attempts++;
    return route.fulfill(
      attempts === 1
        ? { status: 409, json: { error: { message: 'Membership changed. Review and retry.' } } }
        : { json: { id: 'member', roles: ['auditor', 'policy_author'] } },
    );
  });
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: 'Tenant administrator', exact: true }),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Execution traces', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: 'Administration', exact: true }).click();
  await page.getByRole('button', { name: 'Edit roles', exact: true }).click();
  await page.getByLabel('Policy author', { exact: true }).check();
  await page.getByRole('button', { name: 'Save roles', exact: true }).click();
  await expect(page.locator('.role-editor')).toContainText('Membership changed. Review and retry.');
  await expect(page.getByLabel('Policy author', { exact: true })).toBeChecked();
  await page.getByRole('button', { name: 'Save roles', exact: true }).click();
  await expect(page.locator('.role-editor')).toHaveCount(0);
  await expect(page.getByRole('status')).toContainText('Member roles saved.');
  await page.getByRole('tab', { name: 'Invitations', exact: true }).click();
  await page.getByLabel('Google account email').fill('a@@example.com');
  await page.getByRole('button', { name: 'Create invitation' }).click();
  await expect(
    page.getByText('Enter a valid Google account email, such as teammate@example.com.', {
      exact: true,
    }),
  ).toBeVisible();
  await page.getByRole('tab', { name: 'Integration credentials', exact: true }).click();
  await page.getByLabel('Credential name').fill('   ');
  await page.getByRole('button', { name: 'Create credential', exact: true }).click();
  await expect(page.getByText('Enter a name.', { exact: true })).toBeVisible();
});

test('a completed benchmark still warns when its measured outcomes disagree', async ({ page }) => {
  await setup(page, ['auditor']);
  const run = {
    id: 'disagreement-run',
    seed: 42,
    count: 2,
    status: 'completed',
    created_at: '2026-10-23T00:00:00Z',
  };
  await page.route('**/api/v1/tenants/t/benchmarks', (route) =>
    route.fulfill({ json: { items: [run] } }),
  );
  await page.route('**/api/v1/tenants/t/benchmarks/disagreement-run', (route) =>
    route.fulfill({
      json: {
        ...run,
        results: [
          {
            strategy: 'full_sequential',
            profile: 'complete',
            measurement: 'measured_http_engine_only',
            agreement_rate: 0,
            compulsory_violations: 0,
            mean_cost_microrand: 6600,
            p95_ms: 509,
          },
        ],
      },
    }),
  );
  await page.goto('/');
  await page.getByRole('button', { name: 'Benchmarks', exact: true }).click();
  await page.locator('.run-row').click();
  const dialog = page.getByRole('dialog', { name: 'Comparison report' });
  await expect(dialog.getByRole('alert')).toContainText('outcome disagreements');
  await expect(dialog.getByRole('link', { name: 'Export JSON' })).toBeVisible();
  await expect(dialog.getByText('0.0%', { exact: true })).toBeVisible();
});
