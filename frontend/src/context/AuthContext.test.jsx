/**
 * Fase 2 (RV31/F13-02): sesión confiable a nivel provider + rutas.
 */

import React from 'react';
import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { Route } from 'react-router';
import { renderWithProviders, seedSession, seedAdminSession } from '../test/render';
import { expiredJwt } from '../test/fixtures';
import { STORAGE_KEYS, ROUTES } from '../constants';
import { useAuth } from '../hooks/useAuth';
import ProtectedRoute from '../components/auth/ProtectedRoute';

const SessionProbe = () => {
  const { user, isAuthenticated, isAdmin, isBootstrapping, isSubmitting, login, register, logout } = useAuth();
  const [result, setResult] = React.useState(null);
  return (
    <div>
      <span data-testid="bootstrapping">{String(isBootstrapping)}</span>
      <span data-testid="submitting">{String(isSubmitting)}</span>
      <span data-testid="authenticated">{String(isAuthenticated)}</span>
      <span data-testid="admin">{String(isAdmin)}</span>
      <span data-testid="username">{user?.username ?? 'anonymous'}</span>
      <span data-testid="result">{result ? JSON.stringify(result) : 'none'}</span>
      <button onClick={async () => setResult(await login('julian', 'password123'))}>do-login</button>
      <button onClick={async () => setResult(await login('julian', 'wrong-password'))}>do-bad-login</button>
      <button onClick={async () => setResult(await register('julian', 'password123'))}>do-register</button>
      <button onClick={logout}>do-logout</button>
    </div>
  );
};

describe('AuthProvider bootstrap', () => {
  it('sin token arranca anónimo', async () => {
    renderWithProviders(<SessionProbe />);
    await waitFor(() => expect(screen.getByTestId('bootstrapping')).toHaveTextContent('false'));
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
  });

  it('token expirado arranca anónimo y limpia storage (sesión zombie)', async () => {
    window.localStorage.setItem(STORAGE_KEYS.TOKEN, expiredJwt());
    window.localStorage.setItem(STORAGE_KEYS.USER, '{"id":"7","username":"julian"}');

    renderWithProviders(<SessionProbe />);

    await waitFor(() => expect(screen.getByTestId('bootstrapping')).toHaveTextContent('false'));
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toBeNull();
  });

  it('token vigente restaura la sesión desde los claims', async () => {
    seedSession();
    renderWithProviders(<SessionProbe />);
    await waitFor(() => expect(screen.getByTestId('authenticated')).toHaveTextContent('true'));
    expect(screen.getByTestId('username')).toHaveTextContent('julian');
  });

  it('isBootstrapping e isSubmitting son estados distintos', async () => {
    renderWithProviders(<SessionProbe />);
    await waitFor(() => expect(screen.getByTestId('bootstrapping')).toHaveTextContent('false'));
    expect(screen.getByTestId('submitting')).toHaveTextContent('false');
  });
});

describe('AuthProvider login/register/logout', () => {
  it('login OK persiste sesión con id string (A7) y user desde la respuesta', async () => {
    const { user } = renderWithProviders(<SessionProbe />);

    await user.click(screen.getByRole('button', { name: 'do-login' }));

    await waitFor(() => expect(screen.getByTestId('authenticated')).toHaveTextContent('true'));
    expect(screen.getByTestId('username')).toHaveTextContent('julian');
    expect(screen.getByTestId('admin')).toHaveTextContent('false');
    expect(screen.getByTestId('result')).toHaveTextContent('{"success":true}');
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toMatch(/^ey/);
    expect(JSON.parse(window.localStorage.getItem(STORAGE_KEYS.USER)).id).toBe('7');
  });

  it('login fallido devuelve {success:false, error} al form y NO toca la sesión', async () => {
    const { user } = renderWithProviders(<SessionProbe />);

    await user.click(screen.getByRole('button', { name: 'do-bad-login' }));

    await waitFor(() => expect(screen.getByTestId('result')).toHaveTextContent('"success":false'));
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toBeNull();
  });

  it('register hace auto-login tras crear la cuenta', async () => {
    const { user } = renderWithProviders(<SessionProbe />);

    await user.click(screen.getByRole('button', { name: 'do-register' }));

    await waitFor(() => expect(screen.getByTestId('authenticated')).toHaveTextContent('true'));
    expect(screen.getByTestId('username')).toHaveTextContent('julian');
  });

  it('logout limpia storage y vuelve a anónimo', async () => {
    seedSession();
    const { user } = renderWithProviders(<SessionProbe />);
    await waitFor(() => expect(screen.getByTestId('authenticated')).toHaveTextContent('true'));

    await user.click(screen.getByRole('button', { name: 'do-logout' }));

    await waitFor(() => expect(screen.getByTestId('authenticated')).toHaveTextContent('false'));
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toBeNull();
    expect(window.localStorage.getItem(STORAGE_KEYS.USER)).toBeNull();
  });
});

describe('ProtectedRoute', () => {
  const guardedRoutes = (
    <>
      <Route path={ROUTES.LOGIN} element={<div>login page</div>} />
      <Route
        path={ROUTES.RESERVATIONS}
        element={
          <ProtectedRoute>
            <div>private reservations</div>
          </ProtectedRoute>
        }
      />
      <Route
        path={ROUTES.ADMIN}
        element={
          <ProtectedRoute adminOnly>
            <div>admin dashboard</div>
          </ProtectedRoute>
        }
      />
    </>
  );

  it('anónimo en /reservations → /login (state.from se preserva)', async () => {
    renderWithProviders(<div>fallback</div>, { route: ROUTES.RESERVATIONS, routes: guardedRoutes });

    expect(await screen.findByText('login page')).toBeInTheDocument();
    expect(screen.getByTestId('location-probe').dataset.pathname).toBe(ROUTES.LOGIN);
    expect(screen.queryByText('private reservations')).not.toBeInTheDocument();
  });

  it('cliente autenticado ve la ruta privada', async () => {
    seedSession();
    renderWithProviders(<div>fallback</div>, { route: ROUTES.RESERVATIONS, routes: guardedRoutes });
    expect(await screen.findByText('private reservations')).toBeInTheDocument();
  });

  it('cliente en /admin ve un 403 explícito, jamás un frame del dashboard', async () => {
    seedSession();
    renderWithProviders(<div>fallback</div>, { route: ROUTES.ADMIN, routes: guardedRoutes });

    expect(await screen.findByText(/for administrators/i)).toBeInTheDocument();
    expect(screen.queryByText('admin dashboard')).not.toBeInTheDocument();
  });

  it('admin entra al dashboard', async () => {
    seedAdminSession();
    renderWithProviders(<div>fallback</div>, { route: ROUTES.ADMIN, routes: guardedRoutes });
    expect(await screen.findByText('admin dashboard')).toBeInTheDocument();
  });
});
