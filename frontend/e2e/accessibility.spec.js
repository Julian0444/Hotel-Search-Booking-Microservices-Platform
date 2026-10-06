/**
 * Accesibilidad + límites (plan 13 fase 10):
 * - axe en las 6 rutas principales contra el stack real (bar del plan:
 *   0 violaciones serious/critical), en desktop y mobile.
 * - Viewports 320/390 sin overflow horizontal (asserts automatizados de la
 *   verificación manual de fase 8).
 * - Degradación: search-api / hotels-api caídos producen estado útil con
 *   retry y trace_id — no pantalla blanca ni HTML de nginx. Se usa
 *   stop/start para aislar la caída de cada API; el DNS dinámico del gateway
 *   también soporta cambios de IP al recrear. Las pruebas operan en el
 *   proyecto Compose seleccionado por entorno.
 * - Screenshots de superficies estables (evidencia para el plan 12),
 *   enmascarando fechas dinámicas.
 */

import { execSync } from 'node:child_process';
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { REPO_ROOT, API_BASE } from './helpers/env.js';
import { newApiContext, waitForGateway200, searchCatalog } from './helpers/api.js';
import { readAuthState, injectSession } from './helpers/session.js';
import { dateOnlyFromToday, fillBookingDates, isMobileProject, openBookingForm } from './helpers/ui.js';

const compose = (args) =>
  execSync(`docker compose ${args}`, { cwd: REPO_ROOT, stdio: 'pipe', timeout: 120_000 });

const expectNoSeriousViolations = async (page) => {
  const results = await new AxeBuilder({ page }).analyze();
  const serious = results.violations.filter((v) => ['serious', 'critical'].includes(v.impact));
  expect(
    serious.map((v) => `${v.id} (${v.impact}): ${v.nodes.map((n) => n.target.join(' ')).join(' | ')}`),
  ).toEqual([]);
};

test.describe('axe: 0 violaciones serious/critical en las rutas principales', () => {
  test('Home', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Featured stays' })).toBeVisible();
    await expect(page.locator('#main-content h3 a').first()).toBeVisible();
    await expectNoSeriousViolations(page);
  });

  test('Search con resultados', async ({ page }) => {
    await page.goto('/search');
    await expect(page.getByText(/\d+ stays?/)).toBeVisible();
    await expectNoSeriousViolations(page);
  });

  test('Hotel detail', async ({ page }) => {
    const { seedHotel } = readAuthState();
    await page.goto(`/hotels/${seedHotel.id}`);
    await expect(page.getByRole('heading', { level: 1, name: seedHotel.name })).toBeVisible();
    await expectNoSeriousViolations(page);
  });

  test('Login', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByRole('heading', { level: 1, name: 'Welcome back' })).toBeVisible();
    await expectNoSeriousViolations(page);
  });

  test('My reservations (cliente)', async ({ page }) => {
    const { customerSession } = readAuthState();
    await injectSession(page, customerSession);
    await page.goto('/reservations');
    await expect(page.getByRole('heading', { level: 1, name: 'My reservations' })).toBeVisible();
    await expectNoSeriousViolations(page);
  });

  test('Admin dashboard', async ({ page }) => {
    const { admin } = readAuthState();
    await injectSession(page, admin);
    await page.goto('/admin');
    // En mobile la tabla está oculta por CSS (vista de cards): anclar al heading
    await expect(page.getByRole('heading', { name: /Hotels \(\d+\)/ })).toBeVisible();
    await expectNoSeriousViolations(page);
  });
});

test.describe('viewports 320/390: sin overflow horizontal @desktop-only', () => {
  test('todas las rutas caben en el viewport', async ({ page }) => {
    test.setTimeout(180_000);
    const { admin, customerSession, seedHotel } = readAuthState();

    // Orden deliberado: primero rutas anónimas, después cliente, después
    // admin — los init scripts de sesión se acumulan en la page y el último
    // pisa; por eso el loop externo es por RUTA (no por viewport): /login
    // debe visitarse antes de inyectar sesión alguna o redirige a Home.
    const routes = [
      { path: '/', ready: () => page.getByRole('heading', { name: 'Featured stays' }) },
      { path: '/search', ready: () => page.getByText(/\d+ stays?/) },
      { path: `/hotels/${seedHotel.id}`, ready: () => page.getByRole('heading', { level: 1, name: seedHotel.name }) },
      { path: '/login', ready: () => page.getByRole('heading', { level: 1, name: 'Welcome back' }) },
      { path: '/register', ready: () => page.getByRole('heading', { level: 1, name: 'Create your account' }) },
      { path: '/definitely-not-a-page', ready: () => page.getByRole('heading', { level: 1, name: 'That page does not exist' }) },
      { path: '/reservations', session: customerSession, ready: () => page.getByRole('heading', { level: 1, name: 'My reservations' }) },
      { path: '/admin', session: admin, ready: () => page.getByRole('heading', { level: 1, name: 'Admin dashboard' }) },
    ];

    for (const route of routes) {
      if (route.session) await injectSession(page, route.session);
      for (const viewport of [{ width: 320, height: 568 }, { width: 390, height: 844 }]) {
        await page.setViewportSize(viewport);
        await page.goto(route.path);
        await expect(route.ready()).toBeVisible({ timeout: 15_000 });
        const overflow = await page.evaluate(
          () => document.documentElement.scrollWidth - window.innerWidth,
        );
        expect(overflow, `${route.path} @ ${viewport.width}px desborda ${overflow}px`).toBeLessThanOrEqual(0);
      }
    }
  });
});

test.describe('degradación: estado útil, retry y trace_id @desktop-only', () => {
  test.describe.configure({ mode: 'serial' });

  test.afterAll(async () => {
    // Pase lo que pase, el stack queda entero para los specs siguientes
    compose('start search-api hotels-api');
    const api = await newApiContext();
    await waitForGateway200(api, `${API_BASE}/hotels?limit=1&offset=0`);
    const recovered = await searchCatalog(api, '');
    expect(recovered.ok()).toBe(true);
    await api.dispose();
  });

  test('search-api caído: envelope 502 del gateway, no pantalla blanca', async ({ page }) => {
    test.setTimeout(180_000);
    const q = 'zzzzunmatchableresiliencetestzzzz';
    try {
      compose('stop search-api');
      await page.goto(`/search?q=${q}`);
      await expect(page.getByRole('heading', { name: 'Search is unavailable' })).toBeVisible({
        timeout: 30_000,
      });
      const alert = page.getByRole('alert');
      await expect(alert).toContainText('service temporarily unavailable'); // envelope, no HTML de nginx
      await expect(alert).toContainText('Reference:'); // trace_id visible
      await expect(page.getByRole('navigation', { name: 'Primary' })).toBeVisible(); // la app sigue en pie
    } finally {
      compose('start search-api');
    }

    const api = await newApiContext();
    await waitForGateway200(api, `${API_BASE}/search?q=${q}&limit=12&offset=0`);
    await api.dispose();
    await page.getByRole('button', { name: 'Try again' }).click();
    // Recuperado: la misma query rara ahora responde con el empty state real
    await expect(page.getByRole('heading', { name: 'No stays match that search' })).toBeVisible({
      timeout: 30_000,
    });
  });

  test('hotels-api caído: detail degrada con retry y se recupera', async ({ page }) => {
    test.setTimeout(180_000);
    const { seedHotel } = readAuthState();
    try {
      compose('stop hotels-api');
      await page.goto(`/hotels/${seedHotel.id}`);
      await expect(page.getByRole('heading', { name: 'We could not load this stay' })).toBeVisible({
        timeout: 30_000,
      });
      const alert = page.getByRole('alert');
      await expect(alert).toContainText('service temporarily unavailable');
      await expect(alert).toContainText('Reference:');
    } finally {
      compose('start hotels-api');
    }

    const api = await newApiContext();
    await waitForGateway200(api, `${API_BASE}/hotels/${seedHotel.id}`);
    await api.dispose();
    await page.getByRole('button', { name: 'Try again' }).click();
    await expect(page.getByRole('heading', { level: 1, name: seedHotel.name })).toBeVisible({
      timeout: 30_000,
    });
  });
});

test.describe('evidencia visual para el plan 12 (superficies estables)', () => {
  test('screenshots de Home, Search, Detail, Booking y Dashboard', async ({ page }, testInfo) => {
    test.setTimeout(180_000);
    const { admin, seedHotel } = readAuthState();
    const mobile = isMobileProject(testInfo);
    if (mobile) await page.setViewportSize({ width: 390, height: 844 });
    const dir = `e2e-evidence/${testInfo.project.name}`;
    const shoot = (name, options = {}) =>
      page.screenshot({ path: `${dir}/${name}.png`, fullPage: true, animations: 'disabled', ...options });

    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Featured stays' })).toBeVisible();
    await page.waitForLoadState('networkidle');
    await shoot('home');

    await page.goto('/search');
    await expect(page.getByText(/\d+ stays?/)).toBeVisible();
    await page.waitForLoadState('networkidle');
    await shoot('search');

    await page.goto(`/hotels/${seedHotel.id}`);
    await expect(page.getByRole('heading', { level: 1, name: seedHotel.name })).toBeVisible();
    await page.waitForLoadState('networkidle');
    await shoot('hotel-detail');

    // Booking con summary visible; fechas enmascaradas (dinámicas por run)
    const { scope, prefix } = await openBookingForm(page, mobile);
    await fillBookingDates(page, prefix, {
      checkIn: dateOnlyFromToday(30),
      checkOut: dateOnlyFromToday(33),
    });
    await expect(scope.getByText('Total')).toBeVisible();
    await shoot('booking', {
      fullPage: !mobile,
      mask: [page.locator(`#${prefix}-check-in`), page.locator(`#${prefix}-check-out`)],
    });

    await injectSession(page, admin);
    await page.goto('/admin');
    // A 390px la tabla está oculta por CSS (vista de cards): anclar al heading
    await expect(page.getByRole('heading', { name: /Hotels \(\d+\)/ })).toBeVisible();
    await page.waitForLoadState('networkidle');
    await shoot('dashboard');
  });
});
