/**
 * Recorrido Admin (plan 13 fase 10, @desktop-only como permite el plan):
 * sesión env-driven → tabs independientes → self-delete bloqueado → crear
 * hotel → aparece en search (evento RabbitMQ → Solr) → editar → borrar.
 * Las búsquedas usan las URLs normales del huésped, sin cache-busters.
 */

import { test, expect } from '@playwright/test';
import { newApiContext, deleteHotelsByPrefix, waitForSearchIndexed } from './helpers/api.js';
import { readAuthState, injectSession } from './helpers/session.js';
import { attachConsoleGuard } from './helpers/console-guard.js';

const E2E_HOTEL_PREFIX = 'E2ESuites';

const openLastHotelsPage = async (page) => {
  await expect(page.getByRole('table', { name: 'Hotels' })).toBeVisible();
  const pages = page.getByRole('navigation', { name: 'Hotels pages' });
  if (await pages.count()) await pages.getByRole('button', { name: /^Go to page / }).last().click();
};

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
    // Un único término alfanumérico identifica la fixture sin la semántica OR
    // de "E2E Suites <timestamp>", que también encontraba otros hoteles.
    const hotelName = `${E2E_HOTEL_PREFIX}${Date.now()}`;

    // Crear (sin imágenes: la card pública debe caer al fallback local)
    await page.goto('/admin');
    await page.getByRole('link', { name: 'New hotel' }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'New hotel' })).toBeVisible();
    await page.getByLabel('Hotel name').fill(hotelName);
    await page.getByLabel('Address').fill('Av. E2E 1234');
    await page.getByLabel('City').fill('Testville');
    await page.getByLabel('Country').fill('Argentina');
    // El orden público se compara con el catálogo persistido completo.
    await page.getByLabel('Price per night').fill('999');
    await page.getByLabel('Available rooms').fill('3');
    await page.getByLabel('Description').fill('Original description');
    await page.getByRole('combobox', { name: 'Amenities' }).fill('WiFi');
    await page.getByRole('combobox', { name: 'Amenities' }).press('Enter');
    await page.getByRole('button', { name: 'Create hotel' }).click();

    const createdDialog = page.getByRole('dialog', { name: 'Hotel created' });
    await expect(createdDialog).toBeVisible({ timeout: 20_000 });

    // Página pública con fallback local de imagen
    await createdDialog.getByRole('link', { name: 'View hotel' }).click();
    await expect(page.getByRole('heading', { level: 1, name: hotelName })).toBeVisible();
    await expect(page.locator('img[src*="hotel-fallback"]').first()).toBeVisible();

    // Evento → consumer → Solr: esperar usando la URL ordinaria de búsqueda.
    const api = await newApiContext();
    await waitForSearchIndexed(api, { q: hotelName, matchName: hotelName, timeoutMs: 30_000 });
    await api.dispose();
    await page.goto(`/search?q=${encodeURIComponent(hotelName)}`);
    await expect(page.getByRole('link', { name: hotelName })).toBeVisible({ timeout: 15_000 });

    // Editar (el lápiz es un IconButton component={Link} → rol link)
    await page.goto('/admin');
    await openLastHotelsPage(page);
    await page.getByRole('link', { name: `Edit ${hotelName}` }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'Edit hotel' })).toBeVisible();
    await page.getByLabel('Price per night').fill('0');
    await page.getByLabel('Available rooms').fill('0');
    await page.getByLabel('Rating').fill('0');
    await page.getByLabel('Description').fill('');
    await page.getByRole('combobox', { name: 'Amenities' }).fill('');
    await page.getByRole('combobox', { name: 'Amenities' }).press('Backspace');
    // Añadir y luego quitar una imagen real del mismo origen prueba ambos PUT.
    await page.getByLabel('Image URL').fill(new URL('/images/hotel-fallback.svg', page.url()).href);
    await page.getByRole('button', { name: 'Add', exact: true }).click();
    await page.getByRole('button', { name: 'Save changes' }).click();
    const savedDialog = page.getByRole('dialog', { name: 'Changes saved' });
    await expect(savedDialog).toBeVisible({ timeout: 20_000 });
    await savedDialog.getByRole('link', { name: 'View hotel' }).click();
    const publicPath = new URL(page.url()).pathname;
    const id = publicPath.split('/').pop();
    await page.goto(`/admin/hotels/${id}/edit`);
    await expect(page.getByLabel('Price per night')).toHaveValue('0');
    await expect(page.getByLabel('Available rooms')).toHaveValue('0');
    await expect(page.getByLabel('Rating')).toHaveValue('0');
    await expect(page.getByLabel('Description')).toHaveValue('');
    await expect(page.locator('.MuiAutocomplete-tag')).toHaveCount(0);
    await page.getByRole('button', { name: 'Remove image 1' }).click();
    await page.getByRole('button', { name: 'Save changes' }).click();
    await expect(savedDialog).toBeVisible();
    await savedDialog.getByRole('link', { name: 'View hotel' }).click();
    await page.reload();
    await expect(page.locator('img[src*="hotel-fallback"]').first()).toBeVisible();
    await page.goto(`/admin/hotels/${id}/edit`);
    await expect(page.getByLabel('Hotel name')).toHaveValue(hotelName);
    await expect(page.getByRole('button', { name: 'Remove image 1' })).toHaveCount(0);
    // Guardar limpió dirty: salir del formulario sin advertencia.
    await page.getByRole('button', { name: 'Back to dashboard' }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'Admin dashboard' })).toBeVisible();

    // El precio actualizado llega al catálogo visible usando el término habitual.
    await expect(async () => {
      await page.goto(`/search?q=${encodeURIComponent(hotelName)}`);
      await expect(page.getByRole('link', { name: hotelName })).toBeVisible({ timeout: 2000 });
      await expect(page.getByText(/^\$0\s*\/ night$/)).toBeVisible({ timeout: 2000 });
    }).toPass({ timeout: 30_000 });
    await page.goto('/admin');
    await openLastHotelsPage(page);

    // Borrar (deja el stack como estaba; el índice se limpia por el mismo evento)
    await page.getByRole('button', { name: `Delete ${hotelName}` }).click();
    const deleteDialog = page.getByRole('dialog', { name: 'Delete hotel?' });
    await expect(deleteDialog).toContainText(hotelName);
    await deleteDialog.getByRole('button', { name: 'Delete', exact: true }).click();
    await expect(page.getByText(/Hotel .* deleted/)).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole('table', { name: 'Hotels' }).getByText(hotelName)).toHaveCount(0);
    // El mismo término también deja de devolver el hotel tras DELETE.
    await expect(async () => {
      await page.goto(`/search?q=${encodeURIComponent(hotelName)}`);
      await expect(page.getByText(/\d+ stays?/)).toBeVisible({ timeout: 2000 });
      await expect(page.getByRole('link', { name: hotelName, exact: true })).toHaveCount(0, { timeout: 2000 });
    }).toPass({ timeout: 30_000 });

    expect(consoleErrors).toEqual([]);
  });
});
