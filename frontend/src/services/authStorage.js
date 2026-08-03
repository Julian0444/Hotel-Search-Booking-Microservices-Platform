/**
 * Storage de sesión (plan 13 fase 2, RV31): única puerta a localStorage.
 * Antes AuthContext confiaba en cualquier token/user guardado — la "sesión
 * zombie": un token vencido navegaba como logueado hasta el primer 401.
 * Ahora se decodifican los claims y se validan exp/iss/aud LOCALMENTE antes
 * de aceptar una sesión; el backend sigue siendo la autoridad (firma).
 */

import { STORAGE_KEYS } from '../constants';

const EXPECTED_ISSUER = 'users-api';
// La SPA consume users-api y hotels-api: alcanza con que el token declare
// alguna de esas audiencias (el JWT real trae las tres del tokenizer)
const ACCEPTED_AUDIENCES = ['users-api', 'hotels-api'];

/** Decodifica el payload de un JWT sin verificar firma (eso es del backend). */
export const decodeJwtPayload = (token) => {
  const parts = (token || '').split('.');
  if (parts.length !== 3) return null;
  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
    const json = decodeURIComponent(
      atob(padded)
        .split('')
        .map((c) => '%' + c.charCodeAt(0).toString(16).padStart(2, '0'))
        .join(''),
    );
    return JSON.parse(json);
  } catch {
    return null;
  }
};

/** exp vigente + iss propio + audiencia de la plataforma. */
export const areClaimsValid = (claims, nowSeconds = Math.floor(Date.now() / 1000)) => {
  if (!claims || typeof claims !== 'object') return false;
  if (typeof claims.exp !== 'number' || claims.exp <= nowSeconds) return false;
  if (claims.iss !== EXPECTED_ISSUER) return false;
  const aud = Array.isArray(claims.aud) ? claims.aud : [claims.aud];
  return aud.some((a) => ACCEPTED_AUDIENCES.includes(a));
};

export const persistSession = ({ token, user }) => {
  window.localStorage.setItem(STORAGE_KEYS.TOKEN, token);
  window.localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(user));
};

export const clearSession = () => {
  window.localStorage.removeItem(STORAGE_KEYS.TOKEN);
  window.localStorage.removeItem(STORAGE_KEYS.USER);
};

export const readToken = () => window.localStorage.getItem(STORAGE_KEYS.TOKEN);

/**
 * Lee la sesión guardada. Token ausente, malformado, vencido o de otro
 * emisor → null Y storage limpio (arrancar anónimo, no zombie).
 * El user se reconstruye desde los claims (fuente más confiable que la copia
 * en storage) con id SIEMPRE string (A7).
 */
export const readStoredSession = () => {
  const token = readToken();
  if (!token) return null;

  const claims = decodeJwtPayload(token);
  if (!areClaimsValid(claims)) {
    clearSession();
    return null;
  }

  return {
    token,
    user: {
      id: String(claims.user_id),
      username: claims.username,
      tipo: claims.tipo,
    },
  };
};
