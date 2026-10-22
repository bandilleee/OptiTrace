import { test, expect, type Page } from '@playwright/test';

// No intercepted APIs: these journeys exercise the actual Next proxy, Go API,
// local OIDC provider, PostgreSQL authorization, and simulator.
async function signIn(page: Page, identity: string) {
  await page.goto('/');
  await page.getByRole('link', { name: 'Continue with Google' }).click();
  await expect(
    page.getByRole('heading', { name: 'OptiTrace synthetic identity provider' }),
  ).toBeVisible();
  await page.getByRole('link', { name: `${identity}@optitrace.test`, exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Verification playground' })).toBeVisible();
}
async function signOut(page: Page) {
  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(page.getByRole('link', { name: 'Continue with Google' })).toBeVisible();
}
test('real login, both policy profiles, evidence and persisted trace', async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await signIn(page, 'admin');
  for (const profile of ['conditional', 'complete']) {
    await page.getByLabel('Approved policy').selectOption(`policy-${profile}`);
    await page.getByLabel('Observation scenario').selectOption('clear');
    await page.getByRole('button', { name: 'Run verification', exact: true }).click();
    await expect(
      page.locator('.outcome-row').getByRole('heading', { name: 'Verified', exact: true }),
    ).toBeVisible();
    await page.getByLabel('Observation scenario').selectOption('sim_change');
    await page.getByRole('button', { name: 'Run verification', exact: true }).click();
    await expect(
      page
        .locator('.outcome-row')
        .getByRole('heading', { name: 'Additional verification required', exact: true }),
    ).toBeVisible();
    if (profile === 'complete')
      await expect(page.getByText('Compulsory', { exact: true })).toHaveCount(3);
  }
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({
    path: testInfo.outputPath('live-playground-desktop.png'),
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => window.scrollTo(0, 0));
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  );
  await page.screenshot({
    path: testInfo.outputPath('live-playground-mobile.png'),
    fullPage: true,
  });
  await page.getByRole('button', { name: 'Open navigation', exact: true }).click();
  await page.getByRole('button', { name: 'Execution traces', exact: true }).click();
  const row = page.locator('tbody tr').first();
  await row.getByRole('button').click();
  await expect(page.getByRole('heading', { name: 'Trace details' })).toBeVisible();
});

test('durable worker completes a real benchmark and exports its labelled report', async ({
  page,
}, testInfo) => {
  await signIn(page, 'admin');
  await page.getByRole('button', { name: 'Benchmarks', exact: true }).click();
  await page.getByLabel('Observation count').fill('2');
  await page.getByRole('button', { name: 'Launch benchmark' }).click();
  const report = page
    .locator('.panel')
    .filter({ has: page.getByRole('heading', { name: 'Comparison report', exact: true }) });
  await expect(report.getByText('Completed', { exact: true })).toBeVisible({ timeout: 30000 });
  await expect(report.locator('tbody tr')).toHaveCount(20);
  const url = await report.getByRole('link', { name: 'Export JSON' }).getAttribute('href');
  const response = await page.request.get(url!);
  expect(response.status()).toBe(200);
  const payload = await response.json();
  expect(payload.report.manifest.data).toContain('synthetic');
  expect(payload.report.manifest.observation_sha256).toHaveLength(64);
  expect(
    payload.report.results.every(
      (r: { agreement_rate: number; compulsory_violations: number }) =>
        r.agreement_rate === 1 && r.compulsory_violations === 0,
    ),
  ).toBe(true);
  await page.screenshot({ path: testInfo.outputPath('live-benchmark.png'), fullPage: true });
});
test('independent author and reviewer complete the policy approval workflow', async ({ page }) => {
  const name = `Acceptance policy ${Date.now()}`;
  await signIn(page, 'author');
  await page.getByRole('button', { name: 'Policy library', exact: true }).click();
  await page.getByLabel('Policy name', { exact: true }).fill(name);
  await page.getByRole('button', { name: 'Create draft' }).click();
  const authorRow = page.locator('tbody tr').filter({ hasText: name });
  await authorRow.getByRole('button', { name: 'Submit for review' }).click();
  await expect(authorRow.getByText('Submitted', { exact: true })).toBeVisible();
  await expect(authorRow.getByRole('button', { name: 'Approve version' })).toHaveCount(0);
  await signOut(page);
  await signIn(page, 'reviewer');
  await page.getByRole('button', { name: 'Policy library', exact: true }).click();
  const reviewRow = page.locator('tbody tr').filter({ hasText: name });
  await reviewRow.getByRole('button', { name: 'Approve version' }).click();
  await reviewRow.getByRole('button', { name: 'Activate version' }).click();
  await expect(reviewRow.getByText('Active', { exact: true })).toBeVisible();
});
test('admin creates one-time credentials and invitations, auditor remains read-only', async ({
  page,
}) => {
  await signIn(page, 'admin');
  await page.getByRole('button', { name: 'Administration', exact: true }).click();
  await page.getByRole('tab', { name: 'Integration credentials', exact: true }).click();
  const keyName = `Acceptance integration ${Date.now()}`;
  await page.getByLabel('Credential name').fill(keyName);
  await page.getByRole('button', { name: 'Create credential' }).click();
  await expect(page.locator('.secret-box code')).toContainText('ot_');
  await page.getByRole('button', { name: 'Dismiss secret' }).click();
  await expect(page.locator('.secret-box')).toHaveCount(0);
  const keyRow = page.locator('tbody tr').filter({ hasText: keyName });
  await keyRow.getByRole('button', { name: 'Revoke credential' }).click();
  await expect(keyRow).toHaveCount(0);
  await page.getByRole('tab', { name: 'Invitations', exact: true }).click();
  const email = `acceptance-${Date.now()}@optitrace.test`;
  await page.getByLabel('Google account email').fill(email);
  await page.getByRole('button', { name: 'Create invitation' }).click();
  await expect(page.getByText(email, { exact: true })).toBeVisible();
  await signOut(page);
  await signIn(page, 'auditor');
  await expect(page.getByRole('button', { name: 'Run verification', exact: true })).toBeDisabled();
  await page.getByRole('button', { name: 'Administration', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Administration is restricted' })).toBeVisible();
  const session = await (await page.request.get('/api/v1/auth/me')).json();
  const tenant = session.tenants[0].id;
  const denied = await page.request.post(`/api/v1/tenants/${tenant}/evaluations`, {
    headers: {
      Origin: new URL(page.url()).origin,
      'X-CSRF-Token': session.csrf_token,
      'Idempotency-Key': crypto.randomUUID(),
    },
    data: { policy_id: 'policy-conditional', scenario: 'clear', strategy: 'optitrace' },
  });
  expect(denied.status()).toBe(403);
});
