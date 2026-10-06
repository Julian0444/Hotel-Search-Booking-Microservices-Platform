/**
 * Vitest setup (plan 13 fase 0): jsdom + jest-dom + MSW.
 * Un request no mockeado es un ERROR: los tests no pueden depender de red.
 */

import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { cleanup } from '@testing-library/react';
import { server } from './server';

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));

afterEach(() => {
  cleanup();
  server.resetHandlers();
  window.localStorage.clear();
});

afterAll(() => server.close());

// jsdom no implementa matchMedia y MUI useMediaQuery lo consulta
if (!window.matchMedia) {
  window.matchMedia = (query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  });
}

// jsdom define scrollTo pero solo para loguear "Not implemented" — se pisa
// siempre (lo usan el foco por ruta y la paginación)
window.scrollTo = () => {};
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {};
}
