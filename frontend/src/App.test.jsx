/**
 * Fases 8/9 — routing real de App: una ruta lazy muestra el fallback con
 * layout estable (navbar presente, no spinner full-screen) y después la
 * página; al navegar, RouteAnnouncer enfoca el main y el título de la ruta
 * queda en document.title (RouteMeta).
 */

import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';

// NotFound se mockea detrás de un deferred controlable: es la única forma
// determinística de OBSERVAR el fallback de Suspense (los import() reales
// resuelven en el mismo tick en tests)
const gate = vi.hoisted(() => {
  let release;
  const promise = new Promise((resolve) => {
    release = resolve;
  });
  return { promise, release: () => release() };
});

vi.mock('./pages/NotFound', async (importOriginal) => {
  await gate.promise;
  return importOriginal();
});

afterEach(() => {
  window.history.replaceState(null, '', '/');
});

describe('App routing (FE2)', () => {
  it('ruta lazy: fallback con layout estable, después la página', async () => {
    window.history.replaceState(null, '', '/no-such-page');
    render(<App />);

    // Mientras el chunk baja: loader accesible DENTRO del layout completo
    expect(await screen.findByText(/loading page/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /staylux/i })).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toBeInTheDocument();

    gate.release();
    expect(
      await screen.findByRole('heading', { name: /that page does not exist/i }),
    ).toBeInTheDocument();
  });

  it('navegar de Home a Search resuelve el chunk, enfoca el main y titula la ruta', async () => {
    render(<App />);
    const user = userEvent.setup();

    await screen.findByRole('heading', { name: /stays worth returning to/i });

    const nav = screen.getByRole('navigation', { name: /primary/i });
    await user.click(within(nav).getByRole('link', { name: 'Search' }));

    expect(await screen.findByRole('heading', { name: /^search stays$/i })).toBeInTheDocument();

    // RouteAnnouncer: foco al main tras el cambio de ruta; RouteMeta titula
    await waitFor(() => expect(document.getElementById('main-content')).toHaveFocus());
    expect(document.title).toBe('Search stays · StayLux');
  });
});
