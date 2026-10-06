/**
 * Fase 6 (F13-06): historial fiel — canceladas visibles, status del
 * backend, cancelación con invalidación (la card queda como cancelled).
 */

import { describe, expect, it } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { server } from '../test/server';
import { renderWithProviders, seedSession } from '../test/render';
import { reservationFixture, reservationFixtures } from '../test/fixtures';
import MyReservations from './MyReservations';

const renderPage = () => {
  seedSession();
  return renderWithProviders(<MyReservations />, { route: '/reservations' });
};

describe('MyReservations', () => {
  it('los tabs agrupan sin ocultar: counts de upcoming/past/cancelled/all', async () => {
    renderPage();

    // fixtures: 1 upcoming (2027), 1 cancelled, 1 completed (past)
    expect(await screen.findByRole('tab', { name: 'Upcoming (1)' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Past (1)' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Cancelled (1)' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'All (3)' })).toBeInTheDocument();
  });

  it('la tab Cancelled muestra la reserva cancelada con su chip y total del dominio', async () => {
    const { user } = renderPage();

    await user.click(await screen.findByRole('tab', { name: 'Cancelled (1)' }));

    const card = await screen.findByRole('article');
    expect(within(card).getByText('Cancelled')).toBeInTheDocument();
    // total_price 30100 centavos → $301.00 (dinero del dominio, no calculado)
    expect(within(card).getByText('$301.00')).toBeInTheDocument();
    // Sin botón Cancel: el lifecycle no lo permite
    expect(within(card).queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
  });

  it('cancelar: dialog nombra el recurso, mutation invalida y la card pasa a Cancelled', async () => {
    let cancelled = false;
    server.use(
      http.delete(`/api/v1/reservations/${reservationFixture.id}`, () => {
        cancelled = true;
        return new HttpResponse(null, { status: 204 });
      }),
      http.get('/api/v1/users/:userId/reservations', () => {
        const items = cancelled
          ? reservationFixtures.map((r) =>
              r.id === reservationFixture.id ? { ...r, status: 'cancelled' } : r,
            )
          : reservationFixtures;
        return HttpResponse.json({ data: items, meta: { limit: 100, offset: 0 } });
      }),
    );

    const { user } = renderPage();

    // La upcoming confirmada tiene Cancel
    const upcoming = await screen.findByRole('article');
    await user.click(within(upcoming).getByRole('button', { name: 'Cancel' }));

    // El dialog nombra el hotel
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(reservationFixture.hotel_name)).toBeInTheDocument();

    await user.click(within(dialog).getByRole('button', { name: /^cancel reservation$/i }));

    // Tras invalidar, la tab Cancelled pasa a 2 — nada desapareció del historial
    expect(await screen.findByRole('tab', { name: 'Cancelled (2)' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'All (3)' })).toBeInTheDocument();
    expect(cancelled).toBe(true);
  });

  it('una cancelación no permitida explica por qué dentro del dialog', async () => {
    server.use(
      http.delete(`/api/v1/reservations/${reservationFixture.id}`, () =>
        HttpResponse.json(
          { error: { code: 'forbidden', message: 'users can only cancel their own reservations', trace_id: 't-403' } },
          { status: 403 },
        ),
      ),
    );

    const { user } = renderPage();

    const upcoming = await screen.findByRole('article');
    await user.click(within(upcoming).getByRole('button', { name: 'Cancel' }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /^cancel reservation$/i }));

    expect(await within(dialog).findByText(/only cancel their own/i)).toBeInTheDocument();
    // El dialog sigue abierto (el usuario decide) — el fondo queda aria-hidden
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'All (3)', hidden: true })).toBeInTheDocument();
  });

  it('estado vacío con acción hacia la búsqueda', async () => {
    server.use(
      http.get('/api/v1/users/:userId/reservations', () =>
        HttpResponse.json({ data: [], meta: { limit: 100, offset: 0 } }),
      ),
    );

    renderPage();
    expect(await screen.findByText(/no reservations yet/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /search stays/i })).toBeInTheDocument();
  });

  it('error de carga muestra retry, no pantalla blanca', async () => {
    server.use(
      http.get('/api/v1/users/:userId/reservations', () =>
        HttpResponse.json({ error: { code: 'internal', message: 'boom', trace_id: 't-1' } }, { status: 500 }),
      ),
    );

    renderPage();
    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
  });
});
