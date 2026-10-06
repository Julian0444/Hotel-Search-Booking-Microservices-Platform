/**
 * Fase 3 (RV22/F13-03): URL única fuente de verdad, meta.total global,
 * estados empty/error/retry diferenciados.
 */

import { describe, expect, it } from 'vitest';
import { act, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse, delay } from 'msw';
import { server } from '../test/server';
import { renderWithProviders } from '../test/render';
import { hotelFixtures, errorEnvelope, listEnvelope } from '../test/fixtures';
import Search from './Search';

describe('Search page', () => {
  it('hidrata q/page/sort desde la URL y calcula el offset', async () => {
    let seenUrl = null;
    server.use(
      http.get('/api/v1/search', ({ request }) => {
        seenUrl = new URL(request.url);
        return HttpResponse.json(listEnvelope(hotelFixtures.slice(0, 1), { total: 45, limit: 12, offset: 12 }));
      }),
    );

    renderWithProviders(<Search />, { route: '/search?q=bariloche&page=2&sort=price_asc' });

    await waitFor(() => expect(seenUrl).not.toBeNull());
    expect(seenUrl.searchParams.get('q')).toBe('bariloche');
    expect(seenUrl.searchParams.get('offset')).toBe('12');
    expect(seenUrl.searchParams.get('limit')).toBe('12');
    expect(seenUrl.searchParams.get('sort')).toBe('price_asc');

    // Los controles reflejan la URL
    expect(await screen.findByDisplayValue('bariloche')).toBeInTheDocument();
  });

  it('muestra el total del índice (meta.total) y las páginas correctas', async () => {
    server.use(
      http.get('/api/v1/search', () =>
        HttpResponse.json(listEnvelope(hotelFixtures, { total: 45, limit: 12, offset: 0 })),
      ),
    );

    renderWithProviders(<Search />, { route: '/search' });

    // 45 stays aunque data.length sea 3
    expect(await screen.findByText('45')).toBeInTheDocument();
    // 45/12 → 4 páginas
    expect(await screen.findByRole('button', { name: /go to page 4/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /go to page 5/i })).not.toBeInTheDocument();
  });

  it('una nueva búsqueda resetea page a 1 en la URL', async () => {
    const { user } = renderWithProviders(<Search />, { route: '/search?q=hotel&page=3' });

    const input = await screen.findByLabelText(/search stays/i);
    await user.clear(input);
    await user.type(input, 'palacio{Enter}');

    await waitFor(() => {
      const probe = screen.getByTestId('location-probe');
      expect(probe.dataset.search).toContain('q=palacio');
      expect(probe.dataset.search).not.toContain('page=');
    });
  });

  it('término sin resultados → empty state con acción, no error', async () => {
    server.use(
      http.get('/api/v1/search', () => HttpResponse.json(listEnvelope([], { total: 0 }))),
    );

    renderWithProviders(<Search />, { route: '/search?q=xyzzy' });

    expect(await screen.findByText(/no stays match/i)).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('un 502 muestra error con retry y trace_id, sin pantalla blanca', async () => {
    server.use(
      http.get('/api/v1/search', () =>
        HttpResponse.json(errorEnvelope('internal', 'error searching hotels', 'trace-502'), { status: 502 }),
      ),
    );

    renderWithProviders(<Search />, { route: '/search?q=spa' });

    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/search is unavailable/i)).toBeInTheDocument();
    expect(screen.getByText(/trace-502/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
  });

  it('sort desconocido en la URL cae a relevance (no viaja al backend)', async () => {
    let seenSort = 'unset';
    server.use(
      http.get('/api/v1/search', ({ request }) => {
        seenSort = new URL(request.url).searchParams.get('sort');
        return HttpResponse.json(listEnvelope(hotelFixtures));
      }),
    );

    renderWithProviders(<Search />, { route: '/search?sort=drop_table' });

    await waitFor(() => expect(seenSort).toBeNull());
  });
});

it('no corrige la página usando placeholderData de otra búsqueda', async () => {
  server.use(http.get('/api/v1/search', async ({ request }) => {
    const url = new URL(request.url);
    if (url.searchParams.get('q') === 'small') return HttpResponse.json(listEnvelope(hotelFixtures.slice(0, 1), { total: 1 }));
    await delay(50);
    return HttpResponse.json(listEnvelope(hotelFixtures, { total: 45, limit: 12, offset: Number(url.searchParams.get('offset')) }));
  }));
  const { router } = renderWithProviders(<Search />, { route: '/search?q=small' });
  await screen.findByText(hotelFixtures[0].name);
  await act(() => router.navigate('/search?q=large&page=2'));
  await screen.findByText('45');
  expect(screen.getByTestId('location-probe').dataset.search).toContain('page=2');
});
