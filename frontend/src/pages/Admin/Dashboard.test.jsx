/**
 * Fase 7 (RV31/F13-07): queries independientes por tab, self-delete
 * bloqueado con motivo, Services real y read-only.
 */

import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { server } from '../../test/server';
import { renderWithProviders, seedAdminSession } from '../../test/render';
import { errorEnvelope } from '../../test/fixtures';
import Dashboard from './Dashboard';

const renderDashboard = () => {
  seedAdminSession();
  return renderWithProviders(<Dashboard />, { route: '/admin' });
};

describe('Admin Dashboard', () => {
  it('un error en hotels no oculta users: cada panel tiene retry propio', async () => {
    server.use(
      http.get('/api/v1/hotels', () =>
        HttpResponse.json(errorEnvelope('internal', 'mongo down'), { status: 500 }),
      ),
    );

    const { user } = renderDashboard();

    // Tab Hotels: error con retry
    expect(await screen.findByText(/hotels could not be loaded/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();

    // Tab Users sigue funcionando
    await user.click(screen.getByRole('tab', { name: /users/i }));
    const usersTable = await screen.findByRole('table', { name: /users/i });
    expect(within(usersTable).getByText('admin')).toBeInTheDocument();
    expect(within(usersTable).getByText('julian')).toBeInTheDocument();
  });

  it('self-delete: el botón del propio admin está deshabilitado con motivo', async () => {
    const { user } = renderDashboard();

    await user.click(await screen.findByRole('tab', { name: /users/i }));

    // El admin logueado (id 1, username admin) no puede borrarse
    const selfDelete = await screen.findByRole('button', {
      name: /cannot delete your own account/i,
    });
    expect(selfDelete).toBeDisabled();

    // El otro usuario sí es borrable
    expect(screen.getByRole('button', { name: /delete julian/i })).toBeEnabled();
  });

  it('borrar otro usuario pide confirmación nombrándolo y refresca la lista', async () => {
    let deletedId = null;
    server.use(
      http.delete('/api/v1/users/:id', ({ params }) => {
        deletedId = params.id;
        return new HttpResponse(null, { status: 204 });
      }),
    );

    const { user } = renderDashboard();
    await user.click(await screen.findByRole('tab', { name: /users/i }));
    await user.click(await screen.findByRole('button', { name: /delete julian/i }));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('julian')).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    expect(await screen.findByText(/“julian” deleted/i)).toBeInTheDocument();
    expect(deletedId).toBe('7');
  });

  it('Services: estado real {services, summary} del plan 11, read-only', async () => {
    const { user } = renderDashboard();

    await user.click(await screen.findByRole('tab', { name: /services/i }));

    // Summary y latencias reales del fixture
    expect(await screen.findByText(/2\/3 services healthy/i)).toBeInTheDocument();
    expect(screen.getByText('4 ms')).toBeInTheDocument();
    // El search-api caído muestra su estado
    const searchCard = screen.getByRole('article', { name: /search-api status/i });
    expect(within(searchCard).getByText('down')).toBeInTheDocument();

    // Read-only: no existen las acciones mock del panel viejo
    expect(screen.queryByRole('button', { name: /scale/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /restart/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /logs/i })).not.toBeInTheDocument();
    expect(screen.getByText(/read-only observability/i)).toBeInTheDocument();
  });

  it('la paginación de hotels usa meta.total del backend', async () => {
    server.use(
      http.get('/api/v1/hotels', ({ request }) => {
        const url = new URL(request.url);
        const offset = Number(url.searchParams.get('offset')) || 0;
        return HttpResponse.json({
          data: [
            {
              id: `hotel-${offset}`,
              name: `Hotel page ${offset / 10 + 1}`,
              city: 'X',
              country: 'Y',
              price_per_night: 100,
              rating: 4,
              available_rooms: 5,
              check_in_time: '15:00',
              check_out_time: '11:00',
              amenities: [],
              images: [],
            },
          ],
          meta: { total: 25, limit: 10, offset },
        });
      }),
    );

    renderDashboard();

    expect(await screen.findByText('Hotels (25)')).toBeInTheDocument();
    // 25/10 → 3 páginas
    expect(screen.getByRole('button', { name: /go to page 3/i })).toBeInTheDocument();
  });
});
