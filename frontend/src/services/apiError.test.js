import { describe, expect, it } from 'vitest';
import { ApiError, toApiError } from './apiError';

const axiosLikeError = (status, data) => ({
  isAxiosError: true,
  config: {},
  response: { status, data, headers: { 'x-internal': 'secret' } },
  message: `Request failed with status code ${status}`,
});

describe('toApiError', () => {
  it('conserva status, code, message y traceId del envelope', () => {
    const error = toApiError(
      axiosLikeError(409, { error: { code: 'no_availability', message: 'no availability for the requested dates', trace_id: 'abc-123' } }),
    );
    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(409);
    expect(error.code).toBe('no_availability');
    expect(error.message).toBe('no availability for the requested dates');
    expect(error.traceId).toBe('abc-123');
  });

  it('nunca filtra internals cuando el body no es el envelope (HTML de proxy)', () => {
    const error = toApiError(axiosLikeError(502, '<html>nginx bad gateway upstream 10.0.0.7</html>'));
    expect(error.status).toBe(502);
    expect(error.code).toBe('unknown');
    expect(error.message).not.toContain('nginx');
    expect(error.message).not.toContain('10.0.0.7');
    expect(error.traceId).toBeNull();
  });

  it('errores de red quedan como status 0 / code network con mensaje estable', () => {
    const error = toApiError({ isAxiosError: true, message: 'Network Error', config: {} });
    expect(error.status).toBe(0);
    expect(error.code).toBe('network');
    expect(error.message).toMatch(/connection/i);
  });

  it('isClientError distingue 4xx de 5xx/red', () => {
    expect(toApiError(axiosLikeError(404, {})).isClientError).toBe(true);
    expect(toApiError(axiosLikeError(500, {})).isClientError).toBe(false);
    expect(toApiError({ message: 'boom' }).isClientError).toBe(false);
  });

  it('es idempotente sobre un ApiError ya normalizado', () => {
    const original = new ApiError({ status: 401, code: 'unauthorized' });
    expect(toApiError(original)).toBe(original);
  });
});
