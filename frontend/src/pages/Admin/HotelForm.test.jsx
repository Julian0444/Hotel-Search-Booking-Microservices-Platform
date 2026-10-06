/**
 * Fase 7 (F13-08): HotelForm manda el contrato exacto ("HH:mm", price
 * numérico, available_rooms entero), bloquea URLs no http(s) y navega sin
 * temporizadores tras el éxito.
 */

import { describe, expect, it } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import { Route } from 'react-router';
import { http, HttpResponse } from 'msw';
import { server } from '../../test/server';
import { renderWithProviders, seedAdminSession } from '../../test/render';
import HotelForm from './HotelForm';

const renderNewForm = () => {
  seedAdminSession();
  return renderWithProviders(<div>fallback</div>, {
    route: '/admin/hotels/new',
    routes: (
      <>
        <Route path="/admin/hotels/new" element={<HotelForm />} />
        <Route path="/admin" element={<div>dashboard page</div>} />
      </>
    ),
  });
};

const fillRequired = async (user) => {
  await user.type(screen.getByLabelText(/hotel name/i), 'Posada Nueva');
  await user.type(screen.getByLabelText(/address/i), 'Calle 1');
  await user.type(screen.getByLabelText(/^city/i), 'Salta');
  await user.type(screen.getByLabelText(/^country/i), 'Argentina');
  await user.type(screen.getByLabelText(/price per night/i), '120.5');
  await user.type(screen.getByLabelText(/available rooms/i), '8');
};

describe('HotelForm', () => {
  it('envía el payload del contrato final: HH:mm, price numérico, rooms entero', async () => {
    let body = null;
    server.use(
      http.post('/api/v1/admin/hotels', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json({ data: { id: 'new-h-1' } }, { status: 201 });
      }),
    );

    const { user } = renderNewForm();
    await screen.findByRole('heading', { name: /new hotel/i });
    await fillRequired(user);
    await user.click(screen.getByRole('button', { name: /create hotel/i }));

    await screen.findByRole('heading', { name: /hotel created/i });

    expect(body.price_per_night).toBe(120.5);
    expect(body.available_rooms).toBe(8);
    expect(typeof body.available_rooms).toBe('number');
    expect(body.check_in_time).toBe('15:00');
    expect(body.check_out_time).toBe('11:00');
    // Sin campos legacy (el wire es snake_case puro) ni centavos en el payload del hotel
    expect(Object.keys(body).filter((key) => /[A-Z]/.test(key))).toEqual([]);
    expect(body).not.toHaveProperty('total_price');
  });

  it('price negativo o rooms no entero bloquean el submit con summary de errores', async () => {
    const { user } = renderNewForm();
    await screen.findByRole('heading', { name: /new hotel/i });

    await user.type(screen.getByLabelText(/hotel name/i), 'X');
    await user.type(screen.getByLabelText(/address/i), 'X');
    await user.type(screen.getByLabelText(/^city/i), 'X');
    await user.type(screen.getByLabelText(/^country/i), 'X');
    await user.type(screen.getByLabelText(/price per night/i), '-1');
    await user.type(screen.getByLabelText(/available rooms/i), '2.5');

    await user.click(screen.getByRole('button', { name: /create hotel/i }));

    const summary = await screen.findByText(/fix \d+ field/i);
    expect(summary).toBeInTheDocument();
    expect(screen.getAllByText(/price must be 0 or more/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/whole number/i).length).toBeGreaterThan(0);
  });

  it('rechaza URLs de imagen que no sean http(s)', async () => {
    const { user } = renderNewForm();
    await screen.findByRole('heading', { name: /new hotel/i });

    const urlInput = screen.getByLabelText(/image url/i);
    await user.type(urlInput, 'javascript:alert(1)');
    await user.click(screen.getByRole('button', { name: /^add$/i }));

    expect(await screen.findByText(/must start with http/i)).toBeInTheDocument();
    expect(screen.queryByAltText(/hotel image 1/i)).not.toBeInTheDocument();
  });

  it('el éxito ofrece acciones explícitas (View hotel / Back) — sin setTimeout', async () => {
    const { user } = renderNewForm();
    await screen.findByRole('heading', { name: /new hotel/i });
    await fillRequired(user);
    await user.click(screen.getByRole('button', { name: /create hotel/i }));

    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /back to dashboard/i }));

    await waitFor(() => {
      expect(screen.getByText('dashboard page')).toBeInTheDocument();
    });
  });

  it('cancelar con cambios sucios pide confirmación; descartar navega', async () => {
    const { user } = renderNewForm();
    await screen.findByRole('heading', { name: /new hotel/i });

    await user.type(screen.getByLabelText(/hotel name/i), 'Cambio sin guardar');
    await user.click(screen.getByRole('button', { name: /^cancel$/i }));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/discard unsaved changes/i)).toBeInTheDocument();

    await user.click(within(dialog).getByRole('button', { name: /keep editing/i }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.getByLabelText(/hotel name/i)).toHaveValue('Cambio sin guardar');

    await user.click(screen.getByRole('button', { name: /^cancel$/i }));
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /discard changes/i }));

    await waitFor(() => {
      expect(screen.getByText('dashboard page')).toBeInTheDocument();
    });
  });
});
