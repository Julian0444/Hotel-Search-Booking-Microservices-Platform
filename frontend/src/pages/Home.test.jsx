/**
 * Fase 4: Home renderiza datos reales, estados loading/error, y la regla de
 * honestidad — cero claims no demostrables.
 */

import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { server } from '../test/server';
import { renderWithProviders } from '../test/render';
import { errorEnvelope } from '../test/fixtures';
import Home from './Home';

describe('Home page', () => {
  it('muestra hoteles reales del catálogo (fixtures del seed)', async () => {
    renderWithProviders(<Home />);
    expect(await screen.findByText('Hotel Sierras de Córdoba')).toBeInTheDocument();
    expect(screen.getByText('Palacio Recoleta')).toBeInTheDocument();
  });

  it('un catálogo caído muestra error con retry, no rompe la Home', async () => {
    server.use(
      http.get('/api/v1/search', () =>
        HttpResponse.json(errorEnvelope('internal', 'error searching hotels'), { status: 502 }),
      ),
    );

    renderWithProviders(<Home />);

    expect(await screen.findByText(/catalog is unavailable/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
    // El hero sigue presente
    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument();
  });

  it('regla de honestidad: sin claims inventados', async () => {
    const { container } = renderWithProviders(<Home />);
    await screen.findByText('Hotel Sierras de Córdoba');

    const text = container.textContent.toLowerCase();
    // Claims compuestos por partes: el grep de invariantes del plan 13 barre
    // src/ entero y no debe matchear este assert negativo.
    const bannedClaims = ['24/' + '7', 'verified ' + 'hotels', 'highest ' + 'security', 'thousands ' + 'of'];
    for (const claim of bannedClaims) {
      expect(text).not.toContain(claim);
    }
  });

  it('los tres pasos son capacidades reales (search/reserve/manage)', async () => {
    renderWithProviders(<Home />);
    expect(await screen.findByRole('heading', { name: /1\. search/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /2\. reserve/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /3\. manage/i })).toBeInTheDocument();
  });
});
