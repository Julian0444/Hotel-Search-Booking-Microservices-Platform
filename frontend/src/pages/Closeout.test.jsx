import { expect, it } from 'vitest';
import { act, screen, waitFor, within } from '@testing-library/react';
import { Link, Outlet, Route } from 'react-router';
import { http, HttpResponse } from 'msw';
import { renderWithProviders, seedAdminSession, seedSession } from '../test/render';
import { hotelFixture, reservationFixture } from '../test/fixtures';
import { server } from '../test/server';
import Dashboard from './Admin/Dashboard';
import HotelForm from './Admin/HotelForm';
import MyReservations from './MyReservations';
import HotelDetail from './HotelDetail';
import Login from './Login';
import Register from './Register';
import Navbar from '../components/Layout/Navbar';
import ProtectedRoute from '../components/auth/ProtectedRoute';
import SessionExpiredNavigator from '../components/auth/SessionExpiredNavigator';
import { addDaysDateOnly, todayLocal } from '../utils/dateOnly';

// Estos recorridos completos de MUI superan 5s con cobertura en el runner Linux.
// El límite sólo aplica a esas dos secuencias; se conservan todos sus asserts.
const FORM_FLOW_TIMEOUT = 15_000;

for (const kind of ['hotels', 'users']) {
  it(`admin ${kind}: borrar el único elemento de página 2 vuelve a página 1`, async () => {
    seedAdminSession();
    let count = 11;
    server.use(
      http.get(`/api/v1/${kind}`, ({ request }) => {
        const offset = Number(new URL(request.url).searchParams.get('offset'));
        const all = Array.from({ length: count }, (_, i) => ({ ...hotelFixture, id: String(100 + i), name: `Item ${i}`, username: `item_${i}`, tipo: 'cliente' }));
        return HttpResponse.json({ data: all.slice(offset, offset + 10), meta: { total: count, offset, limit: 10 } });
      }),
      http.delete(kind === 'hotels' ? '/api/v1/admin/hotels/:id' : '/api/v1/users/:id', () => {
        count--;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const { user } = renderWithProviders(<Dashboard />);
    if (kind === 'users') await user.click(screen.getByRole('tab', { name: 'Users' }));
    await user.click(await screen.findByRole('button', { name: 'Go to page 2' }));
    await user.click(await screen.findByRole('button', { name: kind === 'hotels' ? 'Delete Item 10' : 'Delete item_10' }));
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Delete', exact: true }));
    await waitFor(() => expect(screen.getAllByText(kind === 'hotels' ? 'Item 0' : 'item_0').length).toBeGreaterThan(0));
    expect(screen.queryByText(kind === 'hotels' ? 'No hotels in the catalog' : 'No users')).not.toBeInTheDocument();
  });
}

it('historial: permite llegar a la reserva 101 cancelada y aclara el alcance de contadores', async () => {
  seedSession();
  const all = Array.from({ length: 101 }, (_, i) => ({ ...reservationFixture, id: `reservation-${i}`, hotel_name: `Stay ${i}`, status: 'cancelled' }));
  server.use(http.get('/api/v1/users/:id/reservations', ({ request }) => {
    const url = new URL(request.url);
    const offset = Number(url.searchParams.get('offset'));
    const limit = Number(url.searchParams.get('limit'));
    return HttpResponse.json({ data: all.slice(offset, offset + limit), meta: { offset, limit } });
  }));
  const { user } = renderWithProviders(<MyReservations />);
  await screen.findByText(/20 bookings loaded/);
  for (let page = 1; page <= 5; page++) {
    await user.click(screen.getByRole('button', { name: 'Load more bookings' }));
    await screen.findByText(new RegExp(`${Math.min(20 * (page + 1), 101)} bookings loaded`));
  }
  await user.click(screen.getByRole('tab', { name: 'Cancelled (101)' }));
  expect(screen.getByText('Stay 100')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Load more bookings' })).not.toBeInTheDocument();
});

it('formulario sucio bloquea enlaces ajenos al formulario y Back del router', async () => {
  seedAdminSession();
  const { user, router } = renderWithProviders(<div>Home destination</div>, {
    route: '/', routes: <Route path="/edit" element={<><Link to="/">Brand home</Link><HotelForm /></>} />,
  });
  await act(() => router.navigate('/edit'));
  await user.type(screen.getByLabelText('Hotel name'), 'Unsaved');
  await user.click(screen.getByRole('link', { name: 'Brand home' }));
  await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Keep editing' }));
  expect(screen.getByLabelText('Hotel name')).toHaveValue('Unsaved');
  await act(() => router.navigate(-1));
  await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Discard changes' }));
  await screen.findByText('Home destination');
});

for (const registration of [false, true]) {
  it(`borrador de reserva vuelve intacto después de ${registration ? 'registro' : 'login'}`, async () => {
    const { user } = renderWithProviders(<div>fallback</div>, { route: `/hotels/${hotelFixture.id}`, routes: <>
      <Route path="/hotels/:id" element={<HotelDetail />} />
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
    </> });
    await screen.findByRole('heading', { name: hotelFixture.name });
    const checkIn = addDaysDateOnly(todayLocal(), 8);
    const checkOut = addDaysDateOnly(todayLocal(), 11);
    await user.type(document.getElementById('panel-check-in'), checkIn);
    await user.type(document.getElementById('panel-check-out'), checkOut);
    const panel = document.getElementById('panel-check-in').closest('form');
    await user.click(within(panel).getByRole('combobox', { name: 'Rooms' }));
    await user.click(screen.getByRole('option', { name: '2', exact: true }));
    await user.click(within(panel).getByRole('combobox', { name: 'Guests' }));
    await user.click(screen.getByRole('option', { name: '3', exact: true }));
    await user.click(within(panel).getByRole('button', { name: 'Sign in to reserve' }));
    if (registration) await user.click(await screen.findByRole('link', { name: 'Create an account' }));
    await user.type(await screen.findByLabelText('Username'), 'julian');
    await user.type(screen.getByLabelText('Password', { exact: true }), 'password123');
    if (registration) await user.type(screen.getByLabelText('Confirm password'), 'password123');
    await user.click(screen.getByRole('button', { name: registration ? 'Create account' : 'Sign in', exact: true }));
    await screen.findByRole('heading', { name: hotelFixture.name });
    expect(document.getElementById('panel-check-in')).toHaveValue(checkIn);
    expect(document.getElementById('panel-check-out')).toHaveValue(checkOut);
    const restored = document.getElementById('panel-check-in').closest('form');
    expect(within(restored).getByRole('combobox', { name: 'Rooms' })).toHaveTextContent('2');
    expect(within(restored).getByRole('combobox', { name: 'Guests' })).toHaveTextContent('3');
  });
}

it('edición envía PUT completo con precio/capacidad/rating cero e imágenes vacías', async () => {
  seedAdminSession();
  let saved;
  server.use(http.put('/api/v1/admin/hotels/:id', async ({ request }) => {
    saved = await request.json();
    return HttpResponse.json({ data: { ...saved, id: hotelFixture.id } });
  }));
  const { user } = renderWithProviders(<div />, { route: `/admin/hotels/${hotelFixture.id}/edit`, routes: <Route path="/admin/hotels/:id/edit" element={<HotelForm />} /> });
  await screen.findByDisplayValue(hotelFixture.name);
  for (const label of ['Price per night', 'Available rooms', 'Rating']) {
    await user.clear(screen.getByLabelText(label));
    await user.type(screen.getByLabelText(label), '0');
  }
  await user.clear(screen.getByLabelText('Description'));
  while (screen.queryByRole('button', { name: 'Remove image 1' })) await user.click(screen.getByRole('button', { name: 'Remove image 1' }));
  await user.click(screen.getByRole('button', { name: 'Save changes' }));
  await screen.findByRole('dialog', { name: 'Changes saved' });
  expect(saved).toMatchObject({ price_per_night: 0, available_rooms: 0, rating: 0, description: '', images: [] });
}, FORM_FLOW_TIMEOUT);

it('Sign out respeta el bloqueo del formulario antes de borrar la sesión', async () => {
  seedAdminSession();
  const { user } = renderWithProviders(<div>Fallback</div>, {
    route: '/admin/edit',
    routes: <Route element={<><SessionExpiredNavigator /><Outlet /></>}>
      <Route path="/admin/edit" element={<ProtectedRoute adminOnly><Navbar /><HotelForm /></ProtectedRoute>} />
      <Route path="/" element={<Navbar />} />
      <Route path="/login" element={<Login />} />
    </Route>,
  });
  await user.type(await screen.findByLabelText('Hotel name'), 'Keep this');
  const signOut = async () => {
    await user.click(await screen.findByRole('button', { name: 'Account menu for admin' }));
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));
  };
  await signOut();
  await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Keep editing' }));
  expect(screen.getByLabelText('Hotel name')).toHaveValue('Keep this');
  expect(window.localStorage.getItem('token')).not.toBeNull();
  await signOut();
  await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Discard changes' }));
  await screen.findByRole('link', { name: 'Sign in' });
  expect(window.localStorage.getItem('token')).toBeNull();
}, FORM_FLOW_TIMEOUT);
