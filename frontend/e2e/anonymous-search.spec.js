/**
 * Recorrido Anonymous (plan 13 fase 10): Home → search → ordenar → abrir
 * hotel → intentar reservar → login con retorno. Corre en desktop + mobile.
 * El orden se compara contra el catálogo persistido completo: las pruebas
 * conservan hoteles con historial y no asumen que sólo existen los seeds.
 */

import { test, expect } from '@playwright/test';
import { DEMO_CUSTOMER } from './helpers/env.js';
import { listHotels, newApiContext } from './helpers/api.js';
import { readAuthState } from './helpers/session.js';
import { attachConsoleGuard } from './helpers/console-guard.js';
import {
  dateOnlyFromToday,
  fillBookingDates,
  isMobileProject,
  openBookingForm,
  pickSelectOption,
  signInViaUi,
} from './helpers/ui.js';

test.describe('anonymous: search, sort y login con retorno', () => {
  test('recorrido completo', async ({ page }, testInfo) => {
    test.setTimeout(120_000);
    const mobile = isMobileProject(testInfo);
    const { seedHotel } = readAuthState();
    const consoleErrors = attachConsoleGuard(page);
    const api = await newApiContext();
    let catalog;
    try {
      catalog = await listHotels(api);
    } finally { await api.dispose(); }
    const sortedNames = (field, direction) => [...catalog]
      .sort((a, b) => direction * (a[field] - b[field]) || a.id.localeCompare(b.id))
      .slice(0, 12).map((hotel) => hotel.name);

    // Home: hero honesto con búsqueda y catálogo real
    await page.goto('/');
    await expect(page.getByRole('heading', { level: 1, name: 'Stays worth returning to' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Featured stays' })).toBeVisible();
    await expect(page.getByRole('link', { name: seedHotel.name })).toBeVisible();

    // Búsqueda desde el hero (submit vacío = catálogo completo)
    await page.getByRole('search').getByRole('button', { name: 'Search' }).click();
    await expect(page).toHaveURL(/\/search/);
    await expect(page.getByRole('heading', { level: 1, name: 'Search stays' })).toBeVisible();
    // Count real desde meta.total, no del largo de la página
    await expect(page.getByText(/\d+ stays?/)).toBeVisible();

    // Sort global: la página debe coincidir con los primeros 12 del catálogo
    // completo, incluso si un hotel más barato comenzó en otra página.
    await page.getByRole('combobox', { name: 'Sort by' }).click();
    await page.getByRole('option', { name: 'Price: low to high' }).click();
    await expect(page).toHaveURL(/sort=price_asc/);
    const cardHeadings = page.locator('#main-content h2 a');
    // El catálogo incluye escrituras recientes de otros casos. Si se perdió
    // su evento al recuperar RabbitMQ, el periódico debe converger en la misma
    // URI normal. Esperar un locator no vuelve a consultar React Query.
    const expectCatalogOrder = async (names) => expect(async () => {
      await page.reload();
      await expect(cardHeadings).toHaveText(names, { timeout: 2000 });
    }).toPass({ timeout: 85_000, intervals: [2000] });
    await expectCatalogOrder(sortedNames('price_per_night', 1));

    // Top rated y desempate estable por ID sobre la misma fuente persistente.
    await page.getByRole('combobox', { name: 'Sort by' }).click();
    await page.getByRole('option', { name: 'Top rated' }).click();
    await expect(page).toHaveURL(/sort=rating_desc/);
    await expectCatalogOrder(sortedNames('rating', -1));

    // Back restaura el estado anterior desde la URL (fuente de verdad)
    await page.goBack();
    await expect(page).toHaveURL(/sort=price_asc/);
    await expect(page.getByRole('combobox', { name: 'Sort by' })).toHaveText('Price: low to high');

    // Buscar el hotel de referencia si fixtures extra lo dejan fuera de página1.
    await page.getByRole('search').getByLabel('Search stays').fill('Sierras');
    await page.getByRole('search').getByRole('button', { name: 'Search', exact: true }).click();
    // Abrir el hotel de referencia
    await page.getByRole('link', { name: seedHotel.name }).click();
    await expect(page.getByRole('heading', { level: 1, name: seedHotel.name })).toBeVisible();
    // Horas "HH:mm" localizadas, jamás RFC3339 (RV21)
    await expect(page.locator('#main-content')).toContainText(/Check-in from \d{1,2}:\d{2} (AM|PM)/);
    await expect(page.locator('#main-content')).not.toContainText('T00:00:00');

    // Intentar reservar sin sesión → CTA honesto y retorno prometido
    const { scope, prefix } = await openBookingForm(page, mobile);
    await fillBookingDates(page, prefix, {
      checkIn: dateOnlyFromToday(30),
      checkOut: dateOnlyFromToday(33),
    });
    await pickSelectOption(page, scope, 'Rooms', '2');
    await pickSelectOption(page, scope, 'Guests', '3');
    await expect(scope.getByText('You will come right back here after signing in.')).toBeVisible();
    await scope.getByRole('button', { name: 'Sign in to reserve' }).click();

    // Login con el cliente demo seedeado (migración 0002) y retorno a la intención
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByRole('heading', { level: 1, name: 'Welcome back' })).toBeVisible();
    await signInViaUi(page, DEMO_CUSTOMER);
    await expect(page).toHaveURL(new RegExp(`/hotels/${seedHotel.id}`));
    await expect(page.getByRole('heading', { level: 1, name: seedHotel.name })).toBeVisible();

    // Con sesión, el CTA pasa a confirmar de verdad
    const after = await openBookingForm(page, mobile);
    await expect(after.scope.getByRole('button', { name: 'Confirm reservation' })).toBeVisible();
    await expect(page.locator(`#${after.prefix}-check-in`)).toHaveValue(dateOnlyFromToday(30));
    await expect(page.locator(`#${after.prefix}-check-out`)).toHaveValue(dateOnlyFromToday(33));
    await expect(after.scope.getByRole('combobox', { name: 'Rooms' })).toHaveText('2');
    await expect(after.scope.getByRole('combobox', { name: 'Guests' })).toHaveText('3');

    expect(consoleErrors).toEqual([]);
  });
});
