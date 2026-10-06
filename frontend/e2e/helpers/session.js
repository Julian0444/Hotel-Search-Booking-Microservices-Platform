/**
 * Sesión inyectada por localStorage (plan 13 fase 10).
 * authStorage valida exp/iss/aud localmente y el backend verifica la firma en
 * cada request → hace falta un JWT REAL (obtenido en global-setup vía
 * /api/v1/login). Claves exactas de STORAGE_KEYS: 'token' y 'user'.
 */

import fs from 'node:fs';
import { fileURLToPath } from 'node:url';

export const AUTH_STATE_FILE = fileURLToPath(new URL('../.auth/state.json', import.meta.url));

/** Estado que dejó global-setup: { admin, customerSession, customerCredentials, seedHotel }. */
export const readAuthState = () => JSON.parse(fs.readFileSync(AUTH_STATE_FILE, 'utf8'));

const sessionFromLogin = (login) => ({
  token: login.token,
  user: { id: String(login.user_id), username: login.username, tipo: login.tipo },
});

/** Debe llamarse ANTES del primer goto de la page. */
export const injectSession = async (page, login) => {
  await page.addInitScript((session) => {
    window.localStorage.setItem('token', session.token);
    window.localStorage.setItem('user', JSON.stringify(session.user));
  }, sessionFromLogin(login));
};
