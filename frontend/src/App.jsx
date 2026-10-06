/**
 * Main Application Component (plan 13).
 * QueryClientProvider envuelve a AuthProvider (el provider invalida queries
 * en logout/expiración). Rutas con React.lazy (FE2): Home queda eager; el
 * admin jamás entra al bundle inicial del visitante.
 */

import { lazy, Suspense, useState } from 'react';
import { createBrowserRouter, createRoutesFromElements, RouterProvider, Outlet, Route } from 'react-router';
import { ThemeProvider, CssBaseline } from '@mui/material';
import { QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from './context/AuthContext';
import { createQueryClient } from './services/queryClient';
import theme from './theme/theme';
import { ROUTES } from './constants';

import { Layout } from './components/Layout';
import ProtectedRoute from './components/auth/ProtectedRoute';
import SessionExpiredNavigator from './components/auth/SessionExpiredNavigator';
import RouteAnnouncer from './components/a11y/RouteAnnouncer';
import RouteFallback from './components/common/RouteFallback';
import Home from './pages/Home';

const Search = lazy(() => import('./pages/Search'));
const HotelDetail = lazy(() => import('./pages/HotelDetail'));
const Login = lazy(() => import('./pages/Login'));
const Register = lazy(() => import('./pages/Register'));
const MyReservations = lazy(() => import('./pages/MyReservations'));
const Dashboard = lazy(() => import('./pages/Admin/Dashboard'));
const HotelForm = lazy(() => import('./pages/Admin/HotelForm'));
const NotFound = lazy(() => import('./pages/NotFound'));

const queryClient = createQueryClient();

const routerRoutes = (
    <Route element={
      <>
        <SessionExpiredNavigator />
        <RouteAnnouncer />
        <Suspense fallback={<RouteFallback />}><Outlet /></Suspense>
      </>
    }>
      {/* Public routes with layout */}
      <Route
        path={ROUTES.HOME}
        element={
          <Layout>
            <Home />
          </Layout>
        }
      />
      <Route
        path={ROUTES.SEARCH}
        element={
          <Layout>
            <Search />
          </Layout>
        }
      />
      <Route
        path="/hotels/:id"
        element={
          <Layout>
            <HotelDetail />
          </Layout>
        }
      />

      {/* Auth routes - no layout */}
      <Route path={ROUTES.LOGIN} element={<Login />} />
      <Route path={ROUTES.REGISTER} element={<Register />} />

      {/* Protected routes */}
      <Route
        path={ROUTES.RESERVATIONS}
        element={
          <ProtectedRoute>
            <Layout>
              <MyReservations />
            </Layout>
          </ProtectedRoute>
        }
      />

      {/* Admin routes */}
      <Route
        path={ROUTES.ADMIN}
        element={
          <ProtectedRoute adminOnly>
            <Layout>
              <Dashboard />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/hotels/new"
        element={
          <ProtectedRoute adminOnly>
            <Layout>
              <HotelForm />
            </Layout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/hotels/:id/edit"
        element={
          <ProtectedRoute adminOnly>
            <Layout>
              <HotelForm />
            </Layout>
          </ProtectedRoute>
        }
      />

      {/* 404 real (fase 8): página útil, no un redirect silencioso */}
      <Route
        path="*"
        element={
          <Layout>
            <NotFound />
          </Layout>
        }
      />
    </Route>
);

function App() {
  const [router] = useState(() => createBrowserRouter(createRoutesFromElements(routerRoutes)));
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

export default App;
