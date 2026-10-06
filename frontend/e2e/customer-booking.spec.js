/**
 * Recorrido Customer (plan 13 fase 10): login → fechas/rooms/guests →
 * reservar → confirmación → My Reservations → cancelar → verla en Cancelled.
 * Corre en desktop + mobile con el cliente FRESCO registrado en global-setup
 * (historial determinístico run a run; los asserts igual se anclan al
 * Confirmation ID para tolerar estado previo del mismo run).
 */

import { test, expect } from '@playwright/test';
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

test.describe('customer: booking completo y cancelación auditable', () => {
  test('recorrido completo', async ({ page }, testInfo) => {
    test.setTimeout(90_000);
    const mobile = isMobileProject(testInfo);
    const { customerCredentials, seedHotel } = readAuthState();
    const consoleErrors = attachConsoleGuard(page);

    // Login por UI con el cliente demo del run
    await page.goto('/login');
    await signInViaUi(page, customerCredentials);
    await expect(page).toHaveURL(/\/$/);

    // Buscar y abrir el hotel de referencia
    const searchBox = page.getByRole('search');
    await searchBox.getByLabel('Search stays').fill('Sierras');
    await searchBox.getByRole('button', { name: 'Search' }).click();
    await expect(page).toHaveURL(/q=Sierras/);
    await page.getByRole('link', { name: seedHotel.name }).click();
    await expect(page.getByRole('heading', { level: 1, name: seedHotel.name })).toBeVisible();

    // Concierge: fechas civiles locales, guests elegidos, total en centavos
    const { scope, prefix } = await openBookingForm(page, mobile);
    await fillBookingDates(page, prefix, {
      checkIn: dateOnlyFromToday(30),
      checkOut: dateOnlyFromToday(33), // 3 noches
    });
    await pickSelectOption(page, scope, 'Guests', '2');
    // Total espejo exacto del backend: round(price*100) * 3 noches * 1 room
    const totalCents = Math.round(seedHotel.price_per_night * 100) * 3;
    const totalLabel = `$${(totalCents / 100).toFixed(2)}`;
    await expect(scope.getByText(/ × 3 nights/)).toBeVisible();
    await expect(scope.getByText(totalLabel)).toBeVisible();

    // Confirmar: POST atómico con Idempotency-Key por intento
    await scope.getByRole('button', { name: 'Confirm reservation' }).click();
    const successDialog = page.getByRole('dialog', { name: 'Reservation confirmed' });
    await expect(successDialog).toBeVisible({ timeout: 20_000 });
    await expect(successDialog).toContainText(seedHotel.name);
    const idLine = await successDialog.getByText(/Confirmation ID:/).textContent();
    const shortId = idLine.match(/Confirmation ID:\s*([A-Z0-9]{8})/)?.[1];
    expect(shortId, `línea de confirmación: "${idLine}"`).toBeTruthy();

    // Historial fiel: la reserva aparece en Upcoming con su ID
    await successDialog.getByRole('link', { name: 'View my reservations' }).click();
    await expect(page.getByRole('heading', { level: 1, name: 'My reservations' })).toBeVisible();
    const reservationCard = page.getByRole('article').filter({ hasText: shortId });
    await expect(reservationCard).toBeVisible();
    await expect(reservationCard).toContainText('3 nights');
    await expect(reservationCard).toContainText('2 guests');
    await expect(reservationCard).toContainText(totalLabel);
    await expect(reservationCard.getByText('Confirmed')).toBeVisible();

    // Cancelar con confirmación que nombra el recurso
    await reservationCard.getByRole('button', { name: 'Cancel', exact: true }).click();
    const cancelDialog = page.getByRole('dialog', { name: 'Cancel this reservation?' });
    await expect(cancelDialog).toContainText(seedHotel.name);
    await cancelDialog.getByRole('button', { name: 'Cancel reservation' }).click();
    await expect(page.getByText('Reservation cancelled — it stays in your history.')).toBeVisible({
      timeout: 15_000,
    });

    // La cancelada NO desaparece: queda auditable bajo Cancelled, sin acción Cancel
    await page.getByRole('tab', { name: /Cancelled \(\d+\)/ }).click();
    const cancelledCard = page.getByRole('article').filter({ hasText: shortId });
    await expect(cancelledCard).toBeVisible();
    await expect(cancelledCard.getByText('Cancelled')).toBeVisible();
    await expect(cancelledCard.getByRole('button', { name: 'Cancel', exact: true })).toHaveCount(0);

    expect(consoleErrors).toEqual([]);
  });
});
