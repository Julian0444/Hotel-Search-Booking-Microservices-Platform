/**
 * Services contra MSW (plan 13, F13-01): un solo shape final del contrato.
 * Estos tests demuestran que los services NO soportan camelCase ni nombres
 * legacy: las aserciones usan exclusivamente los campos snake_case finales.
 */

import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../test/server';
import { hotelFixture, hotelFixtures, reservationFixtures } from '../test/fixtures';
import hotelsService from './hotels.service';
import reservationsService from './reservations.service';
import authService from './auth.service';
import adminService from './admin.service';

describe('hotelsService', () => {
  it('search devuelve la página normalizada con el total real (RV22)', async () => {
    const page = await hotelsService.search({ q: '', offset: 0, limit: 2 });
    expect(page.items).toHaveLength(2);
    expect(page.total).toBe(hotelFixtures.length);
    expect(page.limit).toBe(2);
    // Contrato final: available_rooms/price_per_night/check_in_time
    expect(page.items[0].available_rooms).toBe(20);
    expect(page.items[0].price_per_night).toBe(150.5);
    expect(page.items[0].check_in_time).toBe('15:00');
  });

  it('search manda sort al backend y omite relevance (default del contrato)', async () => {
    const seenSorts = [];
    server.use(
      http.get('/api/v1/search', ({ request }) => {
        seenSorts.push(new URL(request.url).searchParams.get('sort'));
        return HttpResponse.json({ data: [], meta: { total: 0, limit: 12, offset: 0 } });
      }),
    );

    await hotelsService.search({ sort: 'price_asc' });
    await hotelsService.search({ sort: 'relevance' });
    expect(seenSorts).toEqual(['price_asc', null]);
  });

  it('getById devuelve el hotel plano (sin envelope)', async () => {
    const hotel = await hotelsService.getById(hotelFixture.id);
    expect(hotel.name).toBe(hotelFixture.name);
    expect(hotel.check_out_time).toBe('10:00');
  });

  it('checkAvailability postea fechas civiles y devuelve el mapa', async () => {
    let body = null;
    server.use(
      http.post('/api/v1/hotels/availability', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json({ data: { [hotelFixture.id]: true } });
      }),
    );

    const map = await hotelsService.checkAvailability([hotelFixture.id], '2027-03-10', '2027-03-13');
    expect(body).toEqual({ hotel_ids: [hotelFixture.id], check_in: '2027-03-10', check_out: '2027-03-13' });
    expect(map[hotelFixture.id]).toBe(true);
  });
});

describe('reservationsService', () => {
  it('create manda el payload canónico y la Idempotency-Key', async () => {
    let body = null;
    let idempotencyKey = null;
    server.use(
      http.post('/api/v1/reservations', async ({ request }) => {
        body = await request.json();
        idempotencyKey = request.headers.get('idempotency-key');
        return HttpResponse.json({ data: { id: 'r1' } }, { status: 201 });
      }),
    );

    const created = await reservationsService.create(
      { hotel_id: 'h1', check_in: '2027-03-10', check_out: '2027-03-13', num_rooms: 2, num_guests: 3 },
      { idempotencyKey: 'attempt-key-1' },
    );

    expect(created).toEqual({ id: 'r1' });
    expect(idempotencyKey).toBe('attempt-key-1');
    // user_id NO viaja en el body: sale del JWT server-side
    expect(body).toEqual({
      hotel_id: 'h1',
      check_in: '2027-03-10',
      check_out: '2027-03-13',
      num_rooms: 2,
      num_guests: 3,
    });
  });

  it('listByUser devuelve el historial completo, canceladas incluidas', async () => {
    const page = await reservationsService.listByUser('7');
    expect(page.items).toHaveLength(reservationFixtures.length);
    const statuses = page.items.map((r) => r.status);
    expect(statuses).toContain('cancelled');
    // Dinero del dominio: centavos + currency
    expect(page.items[0].total_price).toBe(45150);
    expect(page.items[0].currency).toBe('USD');
  });

  it('cancel resuelve con el 204 sin body', async () => {
    await expect(reservationsService.cancel('a1b2c3')).resolves.toBeUndefined();
  });
});

describe('authService', () => {
  it('login devuelve el LoginResponse con user_id string (A7)', async () => {
    const response = await authService.login('julian', 'password123');
    expect(response.user_id).toBe('7');
    expect(typeof response.token).toBe('string');
    expect(response.tipo).toBe('cliente');
  });

  it('listUsers devuelve página + total para el panel admin', async () => {
    const page = await authService.listUsers({ limit: 1, offset: 0 });
    expect(page.items).toHaveLength(1);
    expect(page.total).toBe(2);
    expect(page.items[0].id).toBe('1');
  });
});

describe('adminService', () => {
  it('getMicroservicesStatus devuelve {services, summary} del plan 11', async () => {
    const status = await adminService.getMicroservicesStatus();
    expect(status.summary.total_services).toBe(3);
    expect(status.services[0].instances[0].latency_ms).toBeDefined();
  });

  it('updateHotel devuelve la representación actualizada (A6)', async () => {
    const hotel = await adminService.updateHotel(hotelFixture.id, { name: 'X' });
    expect(hotel.id).toBe(hotelFixture.id);
  });
});
