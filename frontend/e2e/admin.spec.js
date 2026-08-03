/**
 * Recorrido Admin (plan 13 fase 10, @desktop-only como permite el plan):
 * sesión env-driven → tabs independientes → self-delete bloqueado → crear
 * hotel → aparece en search (evento RabbitMQ → Solr) → editar → borrar.
 * El nombre del hotel es único por run: /api/v1/search se cachea 5 min por
 * URI y un nombre repetido leería el cache viejo.
 */

import { test, expect } from '@playwright/test';
import { newApiContext, deleteHotelsByPrefix, waitForSearchIndexed } from './helpers/api.js';
import { readAuthState, injectSession } from './helpers/session.js';
import { attachConsoleGuard } from './helpers/console-guard.js';

const E2E_HOTEL_PREFIX = 'E2E Suites';

test.describe('admin: panel real y CRUD con sync de búsqueda @desktop-only', () => {
  let auth;

  test.beforeAll(async () => {
    auth = readAuthState();
    // Restos de runs anteriores fallidos no deben afectar los asserts
    const api = await newApiContext();
    await deleteHotelsByPrefix(api, auth.admin.token, E2E_HOTEL_PREFIX);
    await api.dispose();
  });

  test.afterAll(async () => {
    const api = await newApiContext();
    await deleteHotelsByPrefix(api, auth.admin.token, E2E_HOTEL_PREFIX);
    await api.dispose();
  });

  test('tabs independientes, self-delete bloqueado y health read-only', async ({ page }) => {
    const consoleErrors = attachConsoleGuard(page);
    await injectSession(page, auth.admin);

    await page.goto('/admin');
    await expect(page.getByRole('heading', { level: 1, name: 'Admin dashboard' })).toBeVisible();
    await expect(page.getByRole('table', { name: 'Hotels' })).toBeVisible();
    await expect(page.getByRole('heading', { name: /Hotels \(\d+\)/ })).toBeVisible();

    // Users: el admin logueado no puede borrarse y ve el motivo
    await page.getByRole('tab', { name: 'Users' }).click();
    const usersTable = page.getByRole('table', { name: 'Users' });
    await expect(usersTable).toBeVisible();
    await expect(usersTable.getByText('(you)')).toBeVisible();
    const selfDelete = usersTable.getByRole('row', { name: /\(you\)/ }).getByRole('button', {
      name: 'You cannot delete your own account while signed in',
      includeHidden: true,
    });
    await expect(selfDelete).toBeDisabled();

    // Services: observabilidad real read-only, sin control plane ficticio
    await page.getByRole('tab', { name: 'Services' }).click();
    await expect(page.getByText(/\d+\/\d+ services healthy/)).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText('Read-only observability', { exact: false })).toBeVisible();
    await expect(page.getByRole('article', { name: 'users-api status' })).toBeVisible();
    await expect(page.getByRole('button', { name: /restart|scale|logs/i })).toHaveCount(0);

    expect(consoleErrors).toEqual([]);
  });

  test('crear hotel → aparece en search → editar → borrar', async ({ page }) => {
    test.setTimeout(180_000);
    const consoleErrors = attachConsoleGuard(page);
    await injectSession(page, auth.admin);
    const hotelName = `${E2E_HOTEL_PREFIX} ${Date.now()}`;

    // Crear (sin imágenes: la card pública debe caer al fallback local)
    await page.goto('/admin');
    await page.getByRole('link', { name: 'New hotel' }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'New hotel' })).toBeVisible();
    await page.getByLabel('Hotel name').fill(hotelName);
    await page.getByLabel('Address').fill('Av. E2E 1234');
    await page.getByLabel('City').fill('Testville');
    await page.getByLabel('Country').fill('Argentina');
    // Precio alto y sin rating: no interfiere con los asserts de sort del seed
    await page.getByLabel('Price per night').fill('999');
    await page.getByLabel('Available rooms').fill('3');
    await page.getByRole('button', { name: 'Create hotel' }).click();

    const createdDialog = page.getByRole('dialog', { name: 'Hotel created' });
    await expect(createdDialog).toBeVisible({ timeout: 20_000 });

    // Página pública con fallback local de imagen
    await createdDialog.getByRole('link', { name: 'View hotel' }).click();
    await expect(page.getByRole('heading', { level: 1, name: hotelName })).toBeVisible();
    await expect(page.locator('img[src*="hotel-fallback"]').first()).toBeVisible();

    // Evento → consumer → Solr (autoSoftCommit ~1s): poll por API con URIs
    // únicas y recién entonces cargar la página de búsqueda del SPA
    const api = await newApiContext();
    await waitForSearchIndexed(api, { q: hotelName, matchName: hotelName, timeoutMs: 30_000 });
    await api.dispose();
    await page.goto(`/search?q=${encodeURIComponent(hotelName)}`);
    await expect(page.getByRole('link', { name: hotelName })).toBeVisible({ timeout: 15_000 });

    // Editar (el lápiz es un IconButton component={Link} → rol link)
    await page.goto('/admin');
    await page.getByRole('link', { name: `Edit ${hotelName}` }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'Edit hotel' })).toBeVisible();
    await page.getByLabel('Price per night').fill('998');
    await page.getByRole('button', { name: 'Save changes' }).click();
    const savedDialog = page.getByRole('dialog', { name: 'Changes saved' });
    await expect(savedDialog).toBeVisible({ timeout: 20_000 });
    await savedDialog.getByRole('button', { name: 'Back to dashboard' }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'Admin dashboard' })).toBeVisible();

    // Borrar (deja el stack como estaba; el índice se limpia por el mismo evento)
    await page.getByRole('button', { name: `Delete ${hotelName}` }).click();
    const deleteDialog = page.getByRole('dialog', { name: 'Delete hotel?' });
    await expect(deleteDialog).toContainText(hotelName);
    await deleteDialog.getByRole('button', { name: 'Delete', exact: true }).click();
    await expect(page.getByText(/Hotel .* deleted/)).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole('table', { name: 'Hotels' }).getByText(hotelName)).toHaveCount(0);

    expect(consoleErrors).toEqual([]);
  });
});
