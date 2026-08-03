import { describe, expect, it } from 'vitest';
import { STORAGE_KEYS } from '../constants';
import { makeJwt, expiredJwt } from '../test/fixtures';
import { areClaimsValid, decodeJwtPayload, readStoredSession } from './authStorage';

const seedToken = (token) => window.localStorage.setItem(STORAGE_KEYS.TOKEN, token);

describe('decodeJwtPayload', () => {
  it('decodifica el payload base64url', () => {
    const claims = decodeJwtPayload(makeJwt({ username: 'ana', tipo: 'administrador' }));
    expect(claims.username).toBe('ana');
    expect(claims.tipo).toBe('administrador');
    expect(claims.iss).toBe('users-api');
  });

  it('devuelve null con tokens malformados', () => {
    expect(decodeJwtPayload('not-a-jwt')).toBeNull();
    expect(decodeJwtPayload('a.b')).toBeNull();
    expect(decodeJwtPayload('x.%%%.y')).toBeNull();
    expect(decodeJwtPayload('')).toBeNull();
  });
});

describe('areClaimsValid (RV31: exp/iss/aud antes de aceptar sesión)', () => {
  const base = decodeJwtPayload(makeJwt());

  it('acepta claims vigentes del emisor propio', () => {
    expect(areClaimsValid(base)).toBe(true);
  });

  it('rechaza exp vencido', () => {
    expect(areClaimsValid({ ...base, exp: Math.floor(Date.now() / 1000) - 1 })).toBe(false);
    expect(areClaimsValid({ ...base, exp: undefined })).toBe(false);
  });

  it('rechaza otro issuer u otra audiencia', () => {
    expect(areClaimsValid({ ...base, iss: 'evil-api' })).toBe(false);
    expect(areClaimsValid({ ...base, aud: ['payments-api'] })).toBe(false);
  });

  it('acepta aud como string simple (formato JWT válido)', () => {
    expect(areClaimsValid({ ...base, aud: 'hotels-api' })).toBe(true);
  });
});

describe('readStoredSession', () => {
  it('token ausente → anónimo', () => {
    expect(readStoredSession()).toBeNull();
  });

  it('token malformado → anónimo y storage limpio', () => {
    seedToken('garbage');
    window.localStorage.setItem(STORAGE_KEYS.USER, '{"id":"7"}');
    expect(readStoredSession()).toBeNull();
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toBeNull();
    expect(window.localStorage.getItem(STORAGE_KEYS.USER)).toBeNull();
  });

  it('token expirado → anónimo y storage limpio (la sesión zombie de RV31)', () => {
    seedToken(expiredJwt());
    expect(readStoredSession()).toBeNull();
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toBeNull();
  });

  it('token vigente → sesión con user derivado de los claims, id string (A7)', () => {
    seedToken(makeJwt({ user_id: 7, username: 'julian', tipo: 'cliente' }));
    const session = readStoredSession();
    expect(session.user).toEqual({ id: '7', username: 'julian', tipo: 'cliente' });
    expect(typeof session.token).toBe('string');
  });
});
