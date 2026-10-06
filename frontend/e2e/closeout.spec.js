import { test, expect } from '@playwright/test';
import { API_BASE } from './helpers/env.js';
import { newApiContext, apiRegister } from './helpers/api.js';
import { readAuthState, injectSession } from './helpers/session.js';
import { attachConsoleGuard } from './helpers/console-guard.js';
import { dateOnlyFromToday, fillBookingDates, isMobileProject, openBookingForm, pickSelectOption } from './helpers/ui.js';

const hotelPayload = (name) => ({
  name, description: 'E2E fixture', address: 'Test 123', city: 'Testville', country: 'Argentina',
  price_per_night: 999, rating: 0, available_rooms: 1,
  check_in_time: '15:00', check_out_time: '11:00', amenities: [], images: [],
});
const createHotel = async (api, token, name) => {
  const response = await api.post(`${API_BASE}/admin/hotels`, {
    headers: { Authorization: `Bearer ${token}` }, data: hotelPayload(name),
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()).data.id;
};

test('destinos y variantes con/sin tildes desde el buscador', async ({ page }) => {
  const errors = attachConsoleGuard(page);
  await page.goto('/search');
  const searches = [
    ['Mendoza', 'Posada del Vino'], ['Bariloche', 'Refugio del Lago'],
    ['Salta', 'Hostal de la Quebrada'], ['Buenos Aires', 'Palermo Soho Suites'],
    ['Córdoba', 'Hotel Sierras de Córdoba'], ['cordoba', 'Hotel Sierras de Córdoba'],
    ['CORDOBA', 'Hotel Sierras de Córdoba'], ['Sierras', 'Hotel Sierras de Córdoba'],
  ];
  for (const [q, name] of searches) {
    await page.getByRole('search').getByLabel('Search stays').fill(q);
    await page.getByRole('search').getByRole('button', { name: 'Search', exact: true }).click();
    await expect(page.getByRole('link', { name, exact: true })).toBeVisible();
  }
  // País también debe encontrar los cinco hoteles canónicos, aunque otros
  // datos de prueba persistidos lleven los resultados a más de una página.
  const countryPage = (offset) => page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname.endsWith('/search') && url.searchParams.get('q') === 'Argentina'
      && Number(url.searchParams.get('offset')) === offset;
  });
  let responsePromise = countryPage(0);
  await page.getByRole('search').getByLabel('Search stays').fill('Argentina');
  await page.getByRole('search').getByRole('button', { name: 'Search', exact: true }).click();
  const found = new Set();
  let pageNumber = 1;
  for (;;) {
    const response = await responsePromise;
    expect(response.ok()).toBe(true);
    const result = await response.json();
    const names = result.data.map((hotel) => hotel.name);
    await expect(page.locator('h2 a')).toHaveText(names);
    names.forEach((name) => found.add(name));
    if (pageNumber * result.meta.limit >= result.meta.total) break;
    responsePromise = countryPage(pageNumber * result.meta.limit);
    pageNumber++;
    await page.getByRole('navigation', { name: 'Search results pages' })
      .getByRole('button', { name: `Go to page ${pageNumber}`, exact: true }).click();
  }
  for (const name of new Set(searches.map(([, name]) => name))) expect(found.has(name), name).toBe(true);
  expect(errors).toEqual([]);
});

test('registro retorna al hotel con fechas, habitaciones y huéspedes', async ({ page }, testInfo) => {
  const errors = attachConsoleGuard(page);
  const { seedHotel } = readAuthState();
  const mobile = isMobileProject(testInfo);
  await page.goto(`/hotels/${seedHotel.id}`);
  await expect(page.getByRole('heading', { name: seedHotel.name, exact: true })).toBeVisible();
  const { scope, prefix } = await openBookingForm(page, mobile);
  const dates = { checkIn: dateOnlyFromToday(70), checkOut: dateOnlyFromToday(72) };
  await fillBookingDates(page, prefix, dates);
  await pickSelectOption(page, scope, 'Rooms', '2');
  await pickSelectOption(page, scope, 'Guests', '3');
  await scope.getByRole('button', { name: 'Sign in to reserve' }).click();
  await page.getByRole('link', { name: 'Create an account' }).click();
  await expect(page.getByRole('heading', { name: 'Create your account', exact: true })).toBeVisible();
  await page.getByLabel('Username').fill(`signup_${Date.now()}`);
  await page.getByLabel('Password', { exact: true }).fill('E2eSignup123');
  await page.getByLabel('Confirm password').fill('E2eSignup123');
  await page.getByRole('button', { name: 'Create account', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/hotels/${seedHotel.id}$`));
  const restored = await openBookingForm(page, mobile);
  await expect(page.locator(`#${restored.prefix}-check-in`)).toHaveValue(dates.checkIn);
  await expect(page.locator(`#${restored.prefix}-check-out`)).toHaveValue(dates.checkOut);
  await expect(restored.scope.getByRole('combobox', { name: 'Rooms' })).toHaveText('2');
  await expect(restored.scope.getByRole('combobox', { name: 'Guests' })).toHaveText('3');
  expect(errors).toEqual([]);
});

test('último cupo, detalle lleno y cancelación actualizan disponibilidad @desktop-only', async ({ page }) => {
  const errors = attachConsoleGuard(page);
  const { admin, customerSession } = readAuthState();
  const api = await newApiContext();
  const hotelName = `E2E inventory ${Date.now()}`;
  const hotelId = await createHotel(api, admin.token, hotelName);
  const dates = { checkIn: dateOnlyFromToday(80), checkOut: dateOnlyFromToday(82) };
  await injectSession(page, customerSession);
  await page.goto(`/hotels/${hotelId}`);
  await expect(page.getByRole('heading', { name: hotelName })).toBeVisible();
  await fillBookingDates(page, 'panel', dates);
  await page.getByRole('button', { name: 'Confirm reservation' }).click();
  const success = page.getByRole('dialog', { name: 'Reservation confirmed' });
  await expect(success).toBeVisible();
  const confirmation = (await success.getByText(/Confirmation ID:/).textContent()).match(/Confirmation ID:\s*([A-Z0-9]{8})/)[1];
  await success.getByRole('button', { name: 'Keep browsing' }).click();
  await expect(page.getByText('No availability for those dates — try another range.')).toBeVisible();

  // Un cliente que recién abre el detalle obtiene disponibilidad persistente,
  // sin consultar nunca el endpoint admin de reservas por hotel.
  await page.reload();
  await expect(page.getByRole('heading', { name: hotelName })).toBeVisible();
  await fillBookingDates(page, 'panel', dates);
  await expect(page.getByText('No availability for those dates — try another range.')).toBeVisible();
  await page.getByRole('navigation', { name: 'Footer' }).getByRole('link', { name: 'My reservations' }).click();
  const card = page.getByRole('article').filter({ hasText: confirmation });
  await expect(card).toBeVisible();
  await card.getByRole('button', { name: 'Cancel', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel reservation' }).click();
  await expect(page.getByText('Reservation cancelled — it stays in your history.')).toBeVisible();
  await page.getByRole('tab', { name: /Cancelled/ }).click();
  await expect(card.getByText('Cancelled', { exact: true })).toBeVisible();
  await page.goBack();
  await expect(page.getByRole('heading', { name: hotelName })).toBeVisible();
  const availabilityResponse = page.waitForResponse((r) => r.url().endsWith('/hotels/availability') && r.request().method() === 'POST');
  await fillBookingDates(page, 'panel', dates);
  const response = await availabilityResponse;
  expect((await response.json()).data[hotelId]).toBe(true);
  await expect(page.getByText('No availability for those dates — try another range.')).toHaveCount(0);
  // Se conserva el historial; no se fuerza un DELETE que el dominio rechaza.
  await api.dispose();
  expect(errors).toEqual([]);
});

test('formulario sucio protege menú, marca, footer, Cancel y Back @desktop-only', async ({ page }) => {
  const errors = attachConsoleGuard(page);
  const { admin } = readAuthState();
  await injectSession(page, admin);
  await page.goto('/admin');
  await page.getByRole('link', { name: 'New hotel' }).click();
  await page.getByLabel('Hotel name').fill('Unsaved hotel');
  const navigationAttempts = [
    () => page.getByRole('button', { name: 'Cancel', exact: true }).click(),
    () => page.getByRole('button', { name: 'Back to dashboard' }).click(),
    () => page.getByRole('navigation', { name: 'Primary' }).getByRole('link', { name: 'Search', exact: true }).click(),
    () => page.getByRole('link', { name: 'StayLux' }).click(),
    () => page.getByRole('navigation', { name: 'Footer' }).getByRole('link', { name: 'Home' }).click(),
    () => page.evaluate(() => window.history.back()),
    async () => {
      await page.getByRole('button', { name: `Account menu for ${admin.username}` }).click();
      await page.getByRole('menuitem', { name: 'Sign out' }).click();
    },
  ];
  for (const attempt of navigationAttempts) {
    await attempt();
    const dialog = page.getByRole('dialog', { name: 'Discard unsaved changes?' });
    await expect(dialog).toBeVisible();
    await dialog.getByRole('button', { name: 'Keep editing' }).click();
    await expect(page.getByLabel('Hotel name')).toHaveValue('Unsaved hotel');
    await expect(page).toHaveURL(/\/admin\/hotels\/new$/);
  }
  // beforeunload es un diálogo nativo distinto del blocker interno.
  const unload = page.waitForEvent('dialog');
  const reload = page.evaluate(() => window.location.reload());
  const nativeDialog = await unload;
  expect(nativeDialog.type()).toBe('beforeunload');
  await nativeDialog.dismiss();
  await reload;
  await expect(page.getByLabel('Hotel name')).toHaveValue('Unsaved hotel');
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Discard changes' }).click();
  await expect(page).toHaveURL(/\/admin$/);
  await page.getByRole('link', { name: 'New hotel' }).click();
  await page.getByLabel('Hotel name').fill('Sign out draft');
  await page.getByRole('button', { name: `Account menu for ${admin.username}` }).click();
  await page.getByRole('menuitem', { name: 'Sign out' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Discard changes' }).click();
  await expect(page.getByRole('link', { name: 'Sign in', exact: true })).toBeVisible();
  expect(await page.evaluate(() => window.localStorage.getItem('token'))).toBeNull();
  expect(errors).toEqual([]);
});

for (const kind of ['hotels', 'users']) {
  test(`borrar último ${kind} de última página conserva acceso al resto @desktop-only`, async ({ page }) => {
    const errors = attachConsoleGuard(page);
    const { admin } = readAuthState();
    const api = await newApiContext();
    const headers = { Authorization: `Bearer ${admin.token}` };
    const created = [];
    try {
      const response = await api.get(`${API_BASE}/${kind}?limit=10&offset=0`, { headers });
      expect(response.ok()).toBe(true);
      let total = (await response.json()).meta.total;
      do {
        const name = `e2e_page_${Date.now()}_${created.length}`;
        const id = kind === 'hotels' ? await createHotel(api, admin.token, name)
          : (await apiRegister(api, { username: name, password: 'E2ePage123' })).id;
        created.push(String(id));
        total++;
      } while (total < 11 || total % 10 !== 1);
      await injectSession(page, admin);
      await page.goto('/admin');
      if (kind === 'users') await page.getByRole('tab', { name: 'Users' }).click();
      const pages = page.getByRole('navigation', { name: kind === 'hotels' ? 'Hotels pages' : 'Users pages' });
      const lastPage = Math.ceil(total / 10);
      await pages.getByRole('button', { name: `Go to page ${lastPage}`, exact: true }).click();
      const table = page.getByRole('table', { name: kind === 'hotels' ? 'Hotels' : 'Users' });
      await expect(table.locator('tbody tr')).toHaveCount(1);
      await table.getByRole('button', { name: /^Delete / }).click();
      await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click();
      await expect(table.locator('tbody tr')).toHaveCount(10);
      await expect(page.getByText(kind === 'hotels' ? 'No hotels in the catalog' : 'No users')).toHaveCount(0);
      created.pop(); // El último ID recién creado era la última fila, orden estable por ID.
      expect(errors).toEqual([]);
    } finally {
      for (const id of created) {
        const result = await api.delete(`${API_BASE}/${kind === 'hotels' ? 'admin/hotels' : 'users'}/${id}`, { headers });
        expect(result.ok(), await result.text()).toBe(true);
      }
      await api.dispose();
    }
  });
}
