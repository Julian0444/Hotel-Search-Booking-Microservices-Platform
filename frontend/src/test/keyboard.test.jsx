/**
 * Fase 8 — teclado y focus: los dialogs devuelven el foco al trigger al
 * cerrarse, las tabs se operan con flechas + Enter y el drawer mobile cierra
 * con Escape restaurando el foco. (El nombre accesible de los icon buttons
 * lo cubre axe — regla button-name — en a11y.test.jsx.)
 */

import { describe, expect, it } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import { Route } from 'react-router';
import { renderWithProviders, seedSession } from './render';
import { hotelFixture, reservationFixture } from './fixtures';
import { Layout } from '../components/Layout';
import Navbar from '../components/Layout/Navbar';
import MyReservations from '../pages/MyReservations';
import HotelDetail from '../pages/HotelDetail';

describe('MyReservations — dialog de cancelación y tabs', () => {
  const renderPage = () => {
    seedSession();
    return renderWithProviders(
      <Layout>
        <MyReservations />
      </Layout>,
      { route: '/reservations' },
    );
  };

  it('el dialog de cancelar cierra con Escape y devuelve el foco al trigger', async () => {
    const { user } = renderPage();

    const trigger = await screen.findByRole('button', { name: 'Cancel' });
    await user.click(trigger);

    const dialog = await screen.findByRole('dialog', { name: /cancel this reservation/i });
    expect(within(dialog).getByText(reservationFixture.hotel_name)).toBeInTheDocument();

    await user.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it('las tabs se navegan con flechas y activan con Enter', async () => {
    const { user } = renderPage();
    await screen.findByText(reservationFixture.hotel_name);

    const upcomingTab = screen.getByRole('tab', { name: /upcoming/i });
    await user.click(upcomingTab);
    expect(upcomingTab).toHaveAttribute('aria-selected', 'true');

    await user.keyboard('{ArrowRight}');
    const pastTab = screen.getByRole('tab', { name: /past/i });
    expect(pastTab).toHaveFocus();

    await user.keyboard('{Enter}');
    await waitFor(() => expect(pastTab).toHaveAttribute('aria-selected', 'true'));
    expect(upcomingTab).toHaveAttribute('aria-selected', 'false');
  });
});

describe('Navbar — drawer mobile por teclado', () => {
  it('abre desde el botón de menú, atrapa el foco y cierra con Escape devolviéndolo', async () => {
    const { user } = renderWithProviders(<Navbar />);

    const menuButton = screen.getByRole('button', { name: /open navigation menu/i });
    await user.click(menuButton);

    // Con el Modal abierto, el resto de la app queda aria-hidden: los roles
    // visibles son solo los del drawer
    const drawerBrand = await screen.findByRole('link', { name: /staylux/i });
    await user.tab();
    expect(drawerBrand).toHaveFocus();

    await user.keyboard('{Escape}');
    await waitFor(() => expect(menuButton).toHaveFocus());
  });
});

describe('HotelDetail — lightbox de la galería', () => {
  it('abre con View all photos y al cerrar devuelve el foco al trigger', async () => {
    const { user } = renderWithProviders(null, {
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

    const trigger = await screen.findByRole('button', { name: /view all photos/i });
    await user.click(trigger);

    const dialog = await screen.findByRole('dialog', { name: /photos/i });
    await user.click(within(dialog).getByRole('button', { name: /close photo gallery/i }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    await waitFor(() => expect(trigger).toHaveFocus());
  });
});
