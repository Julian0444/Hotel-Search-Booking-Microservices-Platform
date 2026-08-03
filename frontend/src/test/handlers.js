/**
 * Handlers MSW por defecto (plan 13 fase 0): responden el contrato final de
 * los planes 07/11 — envelopes {data, meta} y errores {error:{code,message,
 * trace_id}}. Los tests que necesitan fallas puntuales pisan el handler con
 * server.use(...).
 */

import { http, HttpResponse } from 'msw';
import {
  hotelFixtures,
  userFixtures,
  reservationFixtures,
  loginResponseFixture,
  microservicesFixture,
  listEnvelope,
  errorEnvelope,
} from './fixtures';

// El BASE_URL en tests es el relativo del dev ('/api/v1' sobre localhost)
const API = '/api/v1';

const paginate = (items, url) => {
  const limit = Math.min(Number(url.searchParams.get('limit')) || 20, 100);
  const offset = Number(url.searchParams.get('offset')) || 0;
  return {
    page: items.slice(offset, offset + limit),
    meta: { total: items.length, limit, offset },
  };
};

export const handlers = [
  // --- search-api ---
  http.get(`${API}/search`, ({ request }) => {
    const url = new URL(request.url);
    const q = (url.searchParams.get('q') || '').toLowerCase();
    const sort = url.searchParams.get('sort') || '';

    let items = hotelFixtures.filter(
      (h) => !q || h.name.toLowerCase().includes(q) || h.description.toLowerCase().includes(q),
    );
    if (sort === 'price_asc') items = [...items].sort((a, b) => a.price_per_night - b.price_per_night);
    if (sort === 'price_desc') items = [...items].sort((a, b) => b.price_per_night - a.price_per_night);
    if (sort === 'rating_desc') items = [...items].sort((a, b) => b.rating - a.rating);

    const { page, meta } = paginate(items, url);
    return HttpResponse.json({ data: page, meta });
  }),

  // --- hotels-api ---
  http.get(`${API}/hotels`, ({ request }) => {
    const { page, meta } = paginate(hotelFixtures, new URL(request.url));
    return HttpResponse.json({ data: page, meta });
  }),

  http.get(`${API}/hotels/:id`, ({ params }) => {
    const hotel = hotelFixtures.find((h) => h.id === params.id);
    if (!hotel) {
      return HttpResponse.json(errorEnvelope('hotel_not_found', 'hotel not found'), { status: 404 });
    }
    return HttpResponse.json({ data: hotel });
  }),

  http.post(`${API}/hotels/availability`, async ({ request }) => {
    const body = await request.json();
    const availability = Object.fromEntries((body.hotel_ids || []).map((id) => [id, true]));
    return HttpResponse.json({ data: availability });
  }),

  http.post(`${API}/reservations`, () =>
    HttpResponse.json({ data: { id: 'new-reservation-id-000000' } }, { status: 201 }),
  ),

  http.delete(`${API}/reservations/:id`, () => new HttpResponse(null, { status: 204 })),

  http.get(`${API}/users/:userId/reservations`, ({ request }) => {
    const url = new URL(request.url);
    const limit = Math.min(Number(url.searchParams.get('limit')) || 20, 100);
    const offset = Number(url.searchParams.get('offset')) || 0;
    // El endpoint real no manda total en meta (hotels-api)
    return HttpResponse.json({
      data: reservationFixtures.slice(offset, offset + limit),
      meta: { limit, offset },
    });
  }),

  // --- users-api ---
  http.post(`${API}/login`, async ({ request }) => {
    const { username, password } = await request.json();
    if (username === 'julian' && password === 'password123') {
      return HttpResponse.json({ data: loginResponseFixture });
    }
    return HttpResponse.json(errorEnvelope('unauthorized', 'invalid credentials'), { status: 401 });
  }),

  http.post(`${API}/users`, () => HttpResponse.json({ data: { id: '7' } }, { status: 201 })),

  http.get(`${API}/users`, ({ request }) => {
    const { page, meta } = paginate(userFixtures, new URL(request.url));
    return HttpResponse.json({ data: page, meta });
  }),

  http.delete(`${API}/users/:id`, () => new HttpResponse(null, { status: 204 })),

  // --- admin (hotels-api) ---
  http.post(`${API}/admin/hotels`, () =>
    HttpResponse.json({ data: { id: 'new-hotel-id-0000000000' } }, { status: 201 }),
  ),

  http.put(`${API}/admin/hotels/:id`, ({ params }) => {
    const hotel = hotelFixtures.find((h) => h.id === params.id) || hotelFixtures[0];
    return HttpResponse.json({ data: hotel });
  }),

  http.delete(`${API}/admin/hotels/:id`, () => new HttpResponse(null, { status: 204 })),

  http.get(`${API}/admin/microservices`, () =>
    HttpResponse.json({ data: microservicesFixture }),
  ),
];

export { listEnvelope, errorEnvelope };
