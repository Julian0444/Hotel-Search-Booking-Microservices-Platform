/**
 * Fase 8 — axe a nivel componente: las seis rutas principales renderizadas
 * como en la app (dentro de Layout, salvo Login que trae su propio main) no
 * pueden tener violaciones serious/critical. El contraste y el axe sobre el
 * browser real quedan para Lighthouse (fase 9) y Playwright (fase 10).
 */

import { describe, it } from 'vitest';
import { Route } from 'react-router';
import { screen } from '@testing-library/react';
import { renderWithProviders, seedSession, seedAdminSession } from './render';
import { expectNoSeriousViolations } from './axe';
import { hotelFixture, reservationFixture } from './fixtures';
import { Layout } from '../components/Layout';
import Home from '../pages/Home';
import Search from '../pages/Search';
import HotelDetail from '../pages/HotelDetail';
import Login from '../pages/Login';
import MyReservations from '../pages/MyReservations';
import Dashboard from '../pages/Admin/Dashboard';
import NotFound from '../pages/NotFound';

const AXE_TIMEOUT = 20000;

describe('axe: cero violaciones serious/critical por ruta', () => {
  it('Home', { timeout: AXE_TIMEOUT }, async () => {
    renderWithProviders(
      <Layout>
        <Home />
      </Layout>,
    );
    await screen.findByText(hotelFixture.name);
    await expectNoSeriousViolations();
  });

  it('Search', { timeout: AXE_TIMEOUT }, async () => {
    renderWithProviders(
      <Layout>
        <Search />
      </Layout>,
      { route: '/search' },
    );
    await screen.findByText(hotelFixture.name);
    await expectNoSeriousViolations();
  });

  it('HotelDetail', { timeout: AXE_TIMEOUT }, async () => {
    renderWithProviders(null, {
      route: `/hotels/${hotelFixture.id}`,
      routes: (
        <Route
          path="/hotels/:id"
          element={
            <Layout>
              <HotelDetail />
            </Layout>
          }
        />
      ),
    });
    await screen.findByRole('heading', { level: 1, name: hotelFixture.name });
    await expectNoSeriousViolations();
  });

  it('Login', { timeout: AXE_TIMEOUT }, async () => {
    renderWithProviders(<Login />, { route: '/login' });
    await screen.findByRole('heading', { level: 1, name: /welcome back/i });
    await expectNoSeriousViolations();
  });

  it('MyReservations', { timeout: AXE_TIMEOUT }, async () => {
    seedSession();
    renderWithProviders(
      <Layout>
        <MyReservations />
      </Layout>,
      { route: '/reservations' },
    );
    await screen.findByText(reservationFixture.hotel_name);
    await expectNoSeriousViolations();
  });

  it('Dashboard (admin)', { timeout: AXE_TIMEOUT }, async () => {
    seedAdminSession();
    renderWithProviders(
      <Layout>
        <Dashboard />
      </Layout>,
      { route: '/admin' },
    );
    // La lista admin renderiza tabla (desktop) + cards (mobile): ambas en el DOM
    await screen.findAllByText(hotelFixture.name);
    await expectNoSeriousViolations();
  });

  it('NotFound (404)', { timeout: AXE_TIMEOUT }, async () => {
    renderWithProviders(
      <Layout>
        <NotFound />
      </Layout>,
      { route: '/definitely-not-a-page' },
    );
    await screen.findByRole('heading', { level: 1, name: /that page does not exist/i });
    await expectNoSeriousViolations();
  });
});
