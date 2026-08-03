/**
 * Interacciones de UI compartidas por los specs (plan 13 fase 10).
 * El BookingSidebar (desktop) y el BookingMobileBar coexisten SIEMPRE en el
 * DOM ocultos por CSS → los campos se scopean por prefijo de id
 * (#panel-* / #dialog-*) y el resto por el Dialog mobile.
 */

import { expect } from '@playwright/test';

/** "YYYY-MM-DD" civil construido por componentes LOCALES (regla de dateOnly). */
export const dateOnlyFromToday = (offsetDays) => {
  const date = new Date();
  date.setDate(date.getDate() + offsetDays);
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
};

export const isMobileProject = (testInfo) => testInfo.project.name === 'chromium-mobile';

/**
 * Deja el formulario de booking visible y devuelve su scope + prefijo de ids.
 * En mobile abre el dialog desde la action bar inferior.
 */
export const openBookingForm = async (page, mobile) => {
  if (!mobile) return { scope: page, prefix: 'panel' };
  await page.getByRole('button', { name: 'Reserve' }).click();
  const dialog = page.getByRole('dialog', { name: /^Reserve / });
  await expect(dialog).toBeVisible();
  return { scope: dialog, prefix: 'dialog' };
};

export const fillBookingDates = async (page, prefix, { checkIn, checkOut }) => {
  await page.locator(`#${prefix}-check-in`).fill(checkIn);
  await page.locator(`#${prefix}-check-out`).fill(checkOut);
};

/** Select MUI (no nativo): abrir el combobox y elegir la opción. */
export const pickSelectOption = async (page, scope, name, value) => {
  await scope.getByRole('combobox', { name }).click();
  await page.getByRole('option', { name: value, exact: true }).click();
};

/**
 * Login por UI resiliente al login_limit del gateway (5 r/m burst 3).
 * Un login de UI consume DOS tokens (preflight OPTIONS + POST). Cuando el 429
 * le toca al POST, la UI muestra el mensaje del envelope; cuando le toca al
 * PREFLIGHT, el browser lo trata como fallo CORS y la UI muestra el mensaje
 * de red caída (status 0). Ambos casos son throttling → esperar la ventana
 * (2 tokens ≈ 26s) y reintentar.
 */
export const signInViaUi = async (page, { username, password }, { attempts = 5 } = {}) => {
  for (let attempt = 1; attempt <= attempts; attempt++) {
    await page.getByLabel('Username').fill(username);
    await page.getByLabel('Password', { exact: true }).fill(password);
    await page.getByRole('button', { name: 'Sign in' }).click();

    const outcome = await Promise.race([
      page
        .waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15_000 })
        .then(() => 'ok')
        .catch(() => 'timeout'),
      page
        .getByRole('alert')
        .filter({ hasText: /too many requests|could not reach the server/i })
        .first()
        .waitFor({ timeout: 15_000 })
        .then(() => 'throttled')
        .catch(() => 'timeout'),
    ]);

    if (outcome === 'ok') return;
    if (outcome === 'throttled') {
      await page.waitForTimeout(26_000);
      continue;
    }
    const alertText = await page
      .getByRole('alert')
      .first()
      .textContent()
      .catch(() => '(sin alert visible)');
    throw new Error(`login por UI sin navegación ni rate-limit (intento ${attempt}): ${alertText}`);
  }
  throw new Error('login por UI: rate-limited tras todos los reintentos');
};
