/**
 * Render helper (plan 13 fase 0): envuelve ThemeProvider + QueryClientProvider
 * + AuthProvider + MemoryRouter. Cada test recibe un QueryClient nuevo con
 * retries apagados para que un 500 mockeado falle una sola vez y rápido.
 */

import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, CssBaseline } from '@mui/material';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createMemoryRouter, createRoutesFromElements, RouterProvider, Outlet, Route, useLocation } from 'react-router';
import theme from '../theme/theme';
import { AuthProvider } from '../context/AuthContext';
import { STORAGE_KEYS } from '../constants';
import { makeJwt, clientUserFixture, adminUserFixture } from './fixtures';

export const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 0, refetchOnWindowFocus: false },
      mutations: { retry: false },
    },
  });

/** Deja una sesión válida en localStorage ANTES de montar el AuthProvider. */
export const seedSession = (user = clientUserFixture) => {
  window.localStorage.setItem(STORAGE_KEYS.TOKEN, makeJwt({ user_id: Number(user.id), username: user.username, tipo: user.tipo }));
  window.localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(user));
  return user;
};

export const seedAdminSession = () => seedSession(adminUserFixture);

/** Espía la location actual para asserts de navegación. */
export const LocationProbe = () => {
  const location = useLocation();
  return (
    <output
      data-testid="location-probe"
      data-pathname={location.pathname}
      data-search={location.search}
      hidden
    />
  );
};

export function renderWithProviders(
  ui,
  { route = '/', routes = null, queryClient = createTestQueryClient() } = {},
) {
  const router = createMemoryRouter(createRoutesFromElements(
    <Route element={<><Outlet /><LocationProbe /></>}>
      {routes}
      <Route path="*" element={ui} />
    </Route>,
  ), { initialEntries: [route] });
  const utils = render(
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
  return { ...utils, router, user: userEvent.setup(), queryClient };
}
