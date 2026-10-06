/** Helpers del gateway real: sin cache-busters ni reintentos de autenticación. */

import { request } from '@playwright/test';
import { API_BASE } from './env.js';

export const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export const newApiContext = () => request.newContext({ ignoreHTTPSErrors: true });

export const apiLogin = async (api, { username, password }) => {
  const res = await api.post(`${API_BASE}/login`, { data: { username, password } });
  if (!res.ok()) throw new Error(`login de ${username} falló: ${res.status()} ${await res.text()}`);
  return (await res.json()).data;
};

export const apiRegister = async (api, { username, password }) => {
  const res = await api.post(`${API_BASE}/users`, { data: { username, password } });
  if (!res.ok()) {
    throw new Error(`registro de ${username} falló: ${res.status()} ${await res.text()}`);
  }
  return (await res.json()).data; // { id }
};

/** La misma URL que usa el navegador, sin parámetros para eludir cachés. */
export const searchCatalog = (api, q, offset = 0) =>
  api.get(`${API_BASE}/search`, { params: { q, limit: 12, offset } });

/**
 * El backfill de Solr corre en goroutine DESPUÉS del /readyz de search-api:
 * "stack healthy" no implica "índice poblado". Poll hasta ver resultados
 * (y opcionalmente un hotel puntual, para el sync evento→Solr del admin).
 */
export const waitForSearchIndexed = async (api, { q = '', matchName, timeoutMs = 60_000 } = {}) => {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    let offset = 0;
    while (Date.now() - startedAt < timeoutMs) {
      const res = await searchCatalog(api, q, offset);
      if (!res.ok()) break;
      const { data, meta } = await res.json();
      if (!Array.isArray(data) || data.length === 0) break;
      if (!matchName || data.some((h) => h.name === matchName)) return data;
      offset += data.length;
      if (offset >= meta.total) break;
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
  const hotels = [];
  for (;;) {
    const res = await api.get(`${API_BASE}/hotels`, { params: { limit: 100, offset: hotels.length } });
    if (!res.ok()) throw new Error(`GET /hotels falló: ${res.status()}`);
    const { data, meta } = await res.json();
    hotels.push(...data);
    if (hotels.length >= meta.total) return hotels;
    if (!data.length) throw new Error('GET /hotels devolvió una página vacía antes del total');
  }
};

/** Limpieza de hoteles creados por los E2E (belt & suspenders del spec admin). */
export const deleteHotelsByPrefix = async (api, adminToken, prefix) => {
  const hotels = await listHotels(api);
  for (const hotel of hotels.filter((h) => h.name.startsWith(prefix))) {
    const response = await api.delete(`${API_BASE}/admin/hotels/${hotel.id}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    if (![204, 404].includes(response.status())) {
      throw new Error(`DELETE fixture ${hotel.id} falló: ${response.status()} ${await response.text()}`);
    }
  }
};
