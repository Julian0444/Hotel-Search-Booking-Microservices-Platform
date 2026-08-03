/**
 * Fase 5 (RV21/F13-05): horas legibles, booking con límites, Idempotency-Key
 * por intento y errores del dominio con copy propio.
 */

import { describe, expect, it } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import { Route } from 'react-router';
import { http, HttpResponse } from 'msw';
import { server } from '../test/server';
import { renderWithProviders, seedSession } from '../test/render';
import { hotelFixture, errorEnvelope } from '../test/fixtures';
import { addDaysDateOnly, todayLocal } from '../utils/dateOnly';
import HotelDetail from './HotelDetail';

const renderDetail = () =>
  renderWithProviders(<div>fallback</div>, {
    route: `/hotels/${hotelFixture.id}`,
    routes: (
      <>
        <Route path="/hotels/:id" element={<HotelDetail />} />
        <Route path="/login" element={<div>login page</div>} />
      </>
    ),
  });

const checkInDate = addDaysDateOnly(todayLocal(), 7);
const checkOutDate = addDaysDateOnly(todayLocal(), 10);

const fillDates = async (user) => {
  // El panel desktop y el dialog comparten estado; usamos los inputs del panel
  const checkIn = document.getElementById('panel-check-in');
  const checkOut = document.getElementById('panel-check-out');
  await user.type(checkIn, checkInDate);
  await user.type(checkOut, checkOutDate);
};

describe('HotelDetail', () => {
  it('muestra las horas del contrato "HH:mm" localizadas, nunca RFC3339', async () => {
    renderDetail();

    await screen.findByRole('heading', { name: hotelFixture.name });
    // 15:00 → 3:00 PM / 10:00 → 10:00 AM
    expect(screen.getByText('3:00 PM')).toBeInTheDocument();
    expect(screen.getByText('10:00 AM')).toBeInTheDocument();
    expect(document.body.textContent).not.toMatch(/T\d{2}:\d{2}:\d{2}/);
  });

  it('hotel inexistente → estado 404 útil, no pantalla rota', async () => {
    server.use(
      http.get('/api/v1/hotels/:id', () =>
        HttpResponse.json(errorEnvelope('hotel_not_found', 'hotel not found'), { status: 404 }),
      ),
    );

    renderDetail();
    expect(await screen.findByText(/no longer exists/i)).toBeInTheDocument();
  });

  it('anónimo ve "Sign in to reserve" y el submit navega a login con retorno', async () => {
    const { user } = renderDetail();

    await screen.findByRole('heading', { name: hotelFixture.name });
    const panel = document.getElementById('panel-check-in').closest('form');
    await user.click(within(panel).getByRole('button', { name: /sign in to reserve/i }));

    await waitFor(() => {
      expect(screen.getByTestId('location-probe').dataset.pathname).toBe('/login');
    });
  });

  it('flujo completo: fechas + rooms → total en centavos, submit con Idempotency-Key, confirmación con ID', async () => {
    seedSession();
    let capturedKey = null;
    let capturedBody = null;
    server.use(
      http.post('/api/v1/reservations', async ({ request }) => {
        capturedKey = request.headers.get('idempotency-key');
        capturedBody = await request.json();
        return HttpResponse.json({ data: { id: 'a1b2c3d4e5f60718293a4b5c' } }, { status: 201 });
      }),
    );

    const { user } = renderDetail();
    await screen.findByRole('heading', { name: hotelFixture.name });
    await fillDates(user);

    // Summary visible: 3 noches × 150.50 = 45150 centavos → $451.50
    const panel = document.getElementById('panel-check-in').closest('form');
    expect(await within(panel).findByText('$451.50')).toBeInTheDocument();

    await user.click(within(panel).getByRole('button', { name: /confirm reservation/i }));

    // Confirmación con ID corto copiable y CTA al historial
    expect(await screen.findByRole('heading', { name: /reservation confirmed/i })).toBeInTheDocument();
    expect(screen.getByText('293A4B5C')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /view my reservations/i })).toBeInTheDocument();

    expect(capturedKey).toMatch(/^[0-9a-f-]{36}$/);
    expect(capturedBody).toEqual({
      hotel_id: hotelFixture.id,
      check_in: checkInDate,
      check_out: checkOutDate,
      num_rooms: 1,
      num_guests: 1,
    });
  });

  it('un 409 no_availability explica sin culpar y no bloquea reintentar con otras fechas', async () => {
    seedSession();
    server.use(
      http.post('/api/v1/reservations', () =>
        HttpResponse.json(errorEnvelope('no_availability', 'no availability for the requested dates'), { status: 409 }),
      ),
    );

    const { user } = renderDetail();
    await screen.findByRole('heading', { name: hotelFixture.name });
    await fillDates(user);

    const panel = document.getElementById('panel-check-in').closest('form');
    await user.click(within(panel).getByRole('button', { name: /confirm reservation/i }));

    expect(await screen.findByText(/just sold out/i)).toBeInTheDocument();
    expect(screen.getByText(/nothing was charged/i)).toBeInTheDocument();
  });

  it('un retry de red del MISMO intento conserva la Idempotency-Key', async () => {
    seedSession();
    const seenKeys = [];
    let calls = 0;
    server.use(
      http.post('/api/v1/reservations', ({ request }) => {
        seenKeys.push(request.headers.get('idempotency-key'));
        calls += 1;
        if (calls === 1) {
          return HttpResponse.json(errorEnvelope('internal', 'temporary failure'), { status: 500 });
        }
        return HttpResponse.json({ data: { id: 'r-retry' } }, { status: 201 });
      }),
    );

    const { user } = renderDetail();
    await screen.findByRole('heading', { name: hotelFixture.name });
    await fillDates(user);

    const panel = document.getElementById('panel-check-in').closest('form');
    const confirm = () => within(panel).getByRole('button', { name: /confirm reservation/i });

    await user.click(confirm());
    await screen.findByText(/could not be completed/i);

    await user.click(confirm());
    await screen.findByRole('heading', { name: /reservation confirmed/i });

    expect(seenKeys).toHaveLength(2);
    expect(seenKeys[0]).toBe(seenKeys[1]);
  });
});
