/**
 * Helpers de API contra el gateway real (plan 13 fase 10).
 * Dos gotchas de infra que gobiernan el diseño:
 * - login_limit de nginx: 5 req/min burst 3 por IP → un solo login por rol
 *   por run (global-setup) y retry con espera ante 429.
 * - /api/v1/search se cachea 5 min por URI exacta → todo poll usa un query
 *   param único (probe) para no leer ni envenenar el cache.
 */

import { request } from '@playwright/test';
import { API_BASE } from './env.js';

export const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export const newApiContext = () => request.newContext({ ignoreHTTPSErrors: true });

export const apiLogin = async (api, { username, password }) => {
  for (let attempt = 1; attempt <= 5; attempt++) {
    const res = await api.post(`${API_BASE}/login`, { data: { username, password } });
    if (res.status() === 429) {
      await sleep(13_000); // ventana del login_limit (5 r/m ≈ 1 cada 12s)
      continue;
    }
    if (!res.ok()) {
      throw new Error(`login de ${username} falló: ${res.status()} ${await res.text()}`);
    }
    const body = await res.json();
    return body.data; // { user_id (string), username, token, tipo }
  }
  throw new Error(`login de ${username}: rate-limited tras 5 intentos`);
};

export const apiRegister = async (api, { username, password }) => {
  const res = await api.post(`${API_BASE}/users`, { data: { username, password } });
  if (!res.ok()) {
    throw new Error(`registro de ${username} falló: ${res.status()} ${await res.text()}`);
  }
  return (await res.json()).data; // { id }
};

/** GET /search con URI única (nunca cacheada por el gateway). */
export const searchProbe = (api, q, probe) =>
  api.get(`${API_BASE}/search`, { params: { q, limit: 12, offset: 0, probe } });

/**
 * El backfill de Solr corre en goroutine DESPUÉS del /readyz de search-api:
 * "stack healthy" no implica "índice poblado". Poll hasta ver resultados
 * (y opcionalmente un hotel puntual, para el sync evento→Solr del admin).
 */
export const waitForSearchIndexed = async (api, { q = '', matchName, timeoutMs = 60_000 } = {}) => {
  const startedAt = Date.now();
  let attempt = 0;
  while (Date.now() - startedAt < timeoutMs) {
    const res = await searchProbe(api, q, `probe-${startedAt}-${attempt++}`);
    if (res.ok()) {
      const { data } = await res.json();
      if (Array.isArray(data) && data.length > 0 && (!matchName || data.some((h) => h.name === matchName))) {
        return data;
      }
    }
    await sleep(1500);
  }
  throw new Error(`search no indexado tras ${timeoutMs}ms (q="${q}"${matchName ? `, esperando "${matchName}"` : ''})`);
};

/** Poll genérico hasta que un GET al gateway devuelva 200 (recuperación post-degradación). */
export const waitForGateway200 = async (api, url, { timeoutMs = 90_000 } = {}) => {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    try {
      const res = await api.get(url);
      if (res.ok()) return;
    } catch {
      // gateway aún reconectando; seguir intentando
    }
    await sleep(1500);
  }
  throw new Error(`el gateway no volvió a responder 200 en ${url} tras ${timeoutMs}ms`);
};

export const listHotels = async (api) => {
  const res = await api.get(`${API_BASE}/hotels`, { params: { limit: 50, offset: 0 } });
  if (!res.ok()) throw new Error(`GET /hotels falló: ${res.status()}`);
  return (await res.json()).data;
};

/** Limpieza de hoteles creados por los E2E (belt & suspenders del spec admin). */
export const deleteHotelsByPrefix = async (api, adminToken, prefix) => {
  const hotels = await listHotels(api);
  for (const hotel of hotels.filter((h) => h.name.startsWith(prefix))) {
    await api.delete(`${API_BASE}/admin/hotels/${hotel.id}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
  }
};
