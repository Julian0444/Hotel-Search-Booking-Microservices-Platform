/**
 * Fase 4: Navbar por rol, Footer sin links muertos y ResponsiveImage con fallback.
 */

import { describe, expect, it } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { renderWithProviders, seedSession, seedAdminSession } from '../../test/render';
import Navbar from './Navbar';
import Footer from './Footer';
import ResponsiveImage from '../common/ResponsiveImage';
import { HOTEL_FALLBACK_IMAGE } from '../../constants';

describe('Navbar', () => {
  it('anónimo: Home/Search + sign in/create account, sin admin', async () => {
    renderWithProviders(<Navbar />);
    const nav = await screen.findByRole('navigation', { name: /primary/i });
    expect(within(nav).getByRole('link', { name: 'Home' })).toBeInTheDocument();
    expect(within(nav).getByRole('link', { name: 'Search' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /sign in/i })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /admin/i })).not.toBeInTheDocument();
  });

  it('cliente: reservations visible, admin no', async () => {
    seedSession();
    renderWithProviders(<Navbar />);
    expect(await screen.findByRole('link', { name: /my reservations/i })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();
  });

  it('admin: link al panel presente', async () => {
    seedAdminSession();
    renderWithProviders(<Navbar />);
    expect(await screen.findByRole('link', { name: 'Admin' })).toBeInTheDocument();
  });
});

describe('Footer (regla de honestidad)', () => {
  it('no contiene links muertos ni social falso; el repo es el único externo', () => {
    renderWithProviders(<Footer />);

    const links = screen.getAllByRole('link');
    for (const link of links) {
      expect(link.getAttribute('href')).not.toBe('#');
    }
    const external = links.filter((l) => l.getAttribute('href')?.startsWith('http'));
    expect(external).toHaveLength(1);
    expect(external[0]).toHaveAttribute('rel', expect.stringContaining('noreferrer'));
  });
});

describe('ResponsiveImage', () => {
  it('cae al fallback local tras un error conservando el alt', () => {
    renderWithProviders(<ResponsiveImage src="https://broken.example/x.jpg" alt="Hotel roto" />);

    const img = screen.getByAltText('Hotel roto');
    fireEvent.error(img);

    expect(img).toHaveAttribute('src', HOTEL_FALLBACK_IMAGE);
    expect(img).toHaveAttribute('alt', 'Hotel roto');
  });
});
