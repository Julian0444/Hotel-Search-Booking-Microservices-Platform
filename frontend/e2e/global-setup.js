/**
 * Global setup de los E2E (plan 13 fase 10). Corre UNA vez por run:
 * 1. Espera gateway + índice de Solr poblado (el backfill es asíncrono al
 *    /readyz — "healthy" no implica "indexado").
 * 2. Un solo login de setup por rol:
 *    admin env-driven + un cliente FRESCO registrado por run (historial de
 *    reservas determinístico; no se ensucia al usuario demo seedeado).
 * 3. Persiste tokens y el hotel seed de referencia en e2e/.auth/state.json.
 */

import fs from 'node:fs';
import path from 'node:path';
import { adminCredentials } from './helpers/env.js';
import { newApiContext, apiLogin, apiRegister, waitForSearchIndexed } from './helpers/api.js';
import { AUTH_STATE_FILE } from './helpers/session.js';

export default async function globalSetup() {
  const api = await newApiContext();
  try {
    const hotels = await waitForSearchIndexed(api, {
      q: 'Sierras', matchName: 'Hotel Sierras de Córdoba', timeoutMs: 120_000,
    });
    const seedHotel = hotels.find((h) => h.name === 'Hotel Sierras de Córdoba');

    const admin = await apiLogin(api, adminCredentials());

    const customerCredentials = { username: `e2e_${Date.now()}`, password: 'E2eBooking123' };
    await apiRegister(api, customerCredentials);
    const customerSession = await apiLogin(api, customerCredentials);

    fs.mkdirSync(path.dirname(AUTH_STATE_FILE), { recursive: true });
    fs.writeFileSync(
      AUTH_STATE_FILE,
      JSON.stringify({ admin, customerSession, customerCredentials, seedHotel }, null, 2),
    );
  } finally {
    await api.dispose();
  }
}
