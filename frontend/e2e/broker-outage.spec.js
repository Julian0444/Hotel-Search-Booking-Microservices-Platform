import { execFileSync } from 'node:child_process';
import { test, expect } from '@playwright/test';
import { API_BASE, REPO_ROOT } from './helpers/env.js';
import { newApiContext } from './helpers/api.js';
import { readAuthState, injectSession } from './helpers/session.js';
import { attachConsoleGuard } from './helpers/console-guard.js';

const compose = (...args) => execFileSync('docker', ['compose', ...args], {
  cwd: REPO_ROOT, stdio: 'pipe', timeout: 60_000,
});

test('broker caído: CREATE, PUT y DELETE aplicados tienen éxito visible y búsqueda converge @desktop-only', async ({ page }) => {
  test.setTimeout(300_000);
  const errors = attachConsoleGuard(page);
  const { admin } = readAuthState();
  const api = await newApiContext();
  const headers = { Authorization: `Bearer ${admin.token}` };
  const name = `E2EBroker${Date.now()}`;
  const searchURL = `/search?q=${name}`;
  let id;
  let deleted = false;
  await injectSession(page, admin);

  // Misma URI antes y después de cada operación; sin reindex manual ni probe.
  const converged = async (assertion) => expect(async () => {
    await page.goto(searchURL);
    await assertion();
  }).toPass({ timeout: 85_000, intervals: [2000] });

  try {
    await page.goto(searchURL);
    await expect(page.getByRole('heading', { name: 'No stays match that search' })).toBeVisible();
    compose('stop', 'rabbitmq');

    await page.goto('/admin/hotels/new');
    await page.getByLabel('Hotel name').fill(name);
    await page.getByLabel('Address').fill('Broker test 123');
    await page.getByLabel('City').fill('Testville');
    await page.getByLabel('Country').fill('Argentina');
    await page.getByLabel('Price per night').fill('999');
    await page.getByLabel('Available rooms').fill('2');
    const createResponse = page.waitForResponse((r) => r.request().method() === 'POST' && r.url().endsWith('/admin/hotels'));
    await page.getByRole('button', { name: 'Create hotel' }).click();
    const created = await createResponse;
    expect(created.status()).toBe(201);
    id = (await created.json()).data.id;
    const createdDialog = page.getByRole('dialog', { name: 'Hotel created' });
    await expect(createdDialog).toBeVisible();
    await createdDialog.getByRole('link', { name: 'View hotel' }).click();
    await page.reload();
    await expect(page.getByRole('heading', { name, exact: true })).toBeVisible();
    await converged(() => expect(page.getByRole('link', { name, exact: true })).toBeVisible({ timeout: 2000 }));

    await page.goto(`/admin/hotels/${id}/edit`);
    await expect(page.getByLabel('Hotel name')).toHaveValue(name);
    await page.getByLabel('Price per night').fill('0');
    await page.getByLabel('Available rooms').fill('0');
    await page.getByLabel('Rating').fill('0');
    const updateResponse = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().endsWith(`/admin/hotels/${id}`));
    await page.getByRole('button', { name: 'Save changes' }).click();
    expect((await updateResponse).status()).toBe(200);
    const savedDialog = page.getByRole('dialog', { name: 'Changes saved' });
    await expect(savedDialog).toBeVisible();
    await savedDialog.getByRole('link', { name: 'View hotel' }).click();
    await page.goto(`/admin/hotels/${id}/edit`);
    await expect(page.getByLabel('Price per night')).toHaveValue('0');
    await expect(page.getByLabel('Available rooms')).toHaveValue('0');
    await expect(page.getByLabel('Rating')).toHaveValue('0');
    await converged(async () => {
      await expect(page.getByRole('link', { name, exact: true })).toBeVisible({ timeout: 2000 });
      await expect(page.getByText(/^\$0\s*\/ night$/)).toBeVisible({ timeout: 2000 });
    });

    await page.goto('/admin');
    await expect(page.getByRole('table', { name: 'Hotels' })).toBeVisible();
    const pages = page.getByRole('navigation', { name: 'Hotels pages' });
    if (await pages.count()) await pages.getByRole('button', { name: /^Go to page / }).last().click();
    await page.getByRole('button', { name: `Delete ${name}`, exact: true }).click();
    const deleteResponse = page.waitForResponse((r) => r.request().method() === 'DELETE' && r.url().endsWith(`/admin/hotels/${id}`));
    await page.getByRole('dialog', { name: 'Delete hotel?' }).getByRole('button', { name: 'Delete', exact: true }).click();
    expect((await deleteResponse).status()).toBe(204);
    deleted = true;
    await expect(page.getByText(`Hotel “${name}” deleted`)).toBeVisible();
    expect((await api.get(`${API_BASE}/hotels/${id}`)).status()).toBe(404);
    await converged(() => expect(page.getByRole('heading', { name: 'No stays match that search' })).toBeVisible({ timeout: 2000 }));
    expect(errors).toEqual([]);
  } finally {
    compose('start', 'rabbitmq');
    await expect.poll(() => {
      try {
        compose('exec', '-T', 'rabbitmq', 'rabbitmq-diagnostics', '-q', 'ping');
        return true;
      } catch { return false; }
    }, { timeout: 45_000, intervals: [1000] }).toBe(true);
    if (id && !deleted) {
      const response = await api.delete(`${API_BASE}/admin/hotels/${id}`, { headers });
      expect([204, 404]).toContain(response.status());
    }
    await api.dispose();
  }
});
