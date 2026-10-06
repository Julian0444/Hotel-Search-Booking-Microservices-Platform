/**
 * Interceptores de la instancia Axios (plan 13, F13-02): el 401 de login
 * llega al formulario; el 401 de sesión limpia storage y emite el evento —
 * jamás un hard redirect del navegador.
 */

import { describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../test/server';
import { errorEnvelope, makeJwt } from '../test/fixtures';
import { STORAGE_KEYS } from '../constants';
import api from './api';
import { ApiError } from './apiError';
import { onSessionExpired } from './authEvents';

describe('api interceptors', () => {
  it('adjunta el Bearer token guardado', async () => {
    window.localStorage.setItem(STORAGE_KEYS.TOKEN, makeJwt());
    let seenAuth = null;
    server.use(
      http.get('/api/v1/hotels/abc', ({ request }) => {
        seenAuth = request.headers.get('authorization');
        return HttpResponse.json({ data: { id: 'abc' } });
      }),
    );

    await api.get('/hotels/abc');
    expect(seenAuth).toMatch(/^Bearer ey/);
  });

  it('todo error sale como ApiError normalizado', async () => {
    server.use(
      http.get('/api/v1/hotels/missing', () =>
        HttpResponse.json(errorEnvelope('hotel_not_found', 'hotel not found'), { status: 404 }),
      ),
    );

    const error = await api.get('/hotels/missing').catch((e) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(404);
    expect(error.code).toBe('hotel_not_found');
  });

  it('401 de sesión (con token guardado): limpia storage y emite session-expired', async () => {
    window.localStorage.setItem(STORAGE_KEYS.TOKEN, makeJwt());
    const onExpired = vi.fn();
    const unsubscribe = onSessionExpired(onExpired);
    server.use(
      http.get('/api/v1/users/7/reservations', () =>
        HttpResponse.json(errorEnvelope('unauthorized', 'token expired'), { status: 401 }),
      ),
    );

    await api.get('/users/7/reservations').catch(() => {});

    expect(onExpired).toHaveBeenCalledTimes(1);
    expect(window.localStorage.getItem(STORAGE_KEYS.TOKEN)).toBeNull();
    unsubscribe();
  });

  it('401 del login: NO emite session-expired (credenciales inválidas van al form)', async () => {
    const onExpired = vi.fn();
    const unsubscribe = onSessionExpired(onExpired);

    const error = await api
      .post('/login', { username: 'julian', password: 'wrong-password' })
      .catch((e) => e);

    expect(error.status).toBe(401);
    expect(onExpired).not.toHaveBeenCalled();
    unsubscribe();
  });

  it('401 anónimo (sin token guardado): tampoco emite session-expired', async () => {
    const onExpired = vi.fn();
    const unsubscribe = onSessionExpired(onExpired);
    server.use(
      http.get('/api/v1/users', () =>
        HttpResponse.json(errorEnvelope('unauthorized', 'missing token'), { status: 401 }),
      ),
    );

    await api.get('/users').catch(() => {});
    expect(onExpired).not.toHaveBeenCalled();
    unsubscribe();
  });
});
