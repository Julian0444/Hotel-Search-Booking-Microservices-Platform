/**
 * Recorrido Anonymous (plan 13 fase 10): Home → search → ordenar → abrir
 * hotel → intentar reservar → login con retorno. Corre en desktop + mobile.
 * La paginación real no se cubre acá: el seed (5 hoteles) no supera una
 * página de 12 — queda cubierta por los tests unitarios con MSW, como
 * contempla el plan.
 */

import { test, expect } from '@playwright/test';
import { DEMO_CUSTOMER } from './helpers/env.js';
import { readAuthState } from './helpers/session.js';
import { attachConsoleGuard } from './helpers/console-guard.js';
import {
  dateOnlyFromToday,
  fillBookingDates,
  isMobileProject,
  openBookingForm,
  signInViaUi,
} from './helpers/ui.js';

test.describe('anonymous: search, sort y login con retorno', () => {
  test('recorrido completo', async ({ page }, testInfo) => {
    test.setTimeout(240_000); // absorbe esperas de rate-limit del login
    const mobile = isMobileProject(testInfo);
    const { seedHotel } = readAuthState();
    // El 429 del login_limit es comportamiento deliberado del gateway y el
    // helper de login lo maneja con espera. Un 429 en el PREFLIGHT aparece
    // como error CORS/red en consola — mismo throttling, no debe tirar el guard.
    const consoleErrors = attachConsoleGuard(page, {
      allow: [/status of 429/, /ERR_FAILED/, /blocked by CORS policy/],
    });

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

    // Sort global (backend/Solr, no orden local): el más barato primero
    await page.getByRole('combobox', { name: 'Sort by' }).click();
    await page.getByRole('option', { name: 'Price: low to high' }).click();
    await expect(page).toHaveURL(/sort=price_asc/);
    const cardHeadings = page.locator('#main-content h2 a');
    await expect(cardHeadings.first()).toHaveText('Hostal de la Quebrada');

    // Top rated: el mejor rating del seed primero
    await page.getByRole('combobox', { name: 'Sort by' }).click();
    await page.getByRole('option', { name: 'Top rated' }).click();
    await expect(page).toHaveURL(/sort=rating_desc/);
    await expect(cardHeadings.first()).toHaveText('Refugio del Lago');

    // Back restaura el estado anterior desde la URL (fuente de verdad)
    await page.goBack();
    await expect(page).toHaveURL(/sort=price_asc/);
    await expect(page.getByRole('combobox', { name: 'Sort by' })).toHaveText('Price: low to high');

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

    expect(consoleErrors).toEqual([]);
  });
});
