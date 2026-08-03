/**
 * ApiError (plan 13, F13-01): el único tipo de error que sale de services/.
 * Conserva status, code, message y traceId del envelope estándar
 * {error:{code,message,trace_id}} y NUNCA imprime internals del response.
 */

const FALLBACK_MESSAGES = {
  0: 'Could not reach the server. Check your connection and try again.',
  400: 'The request was invalid.',
  401: 'You need to sign in to do that.',
  403: 'You do not have permission to do that.',
  404: 'We could not find what you were looking for.',
  409: 'That conflicts with the current state. Refresh and try again.',
  429: 'Too many requests. Wait a moment and try again.',
};

export class ApiError extends Error {
  constructor({ status = 0, code = 'unknown', message, traceId = null } = {}) {
    super(message || FALLBACK_MESSAGES[status] || 'Something went wrong. Please try again.');
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.traceId = traceId;
  }

  /** 4xx: reintentar automáticamente no tiene sentido. */
  get isClientError() {
    return this.status >= 400 && this.status < 500;
  }
}

/**
 * Normaliza cualquier error de Axios a ApiError. Un body que no es el
 * envelope estándar (p. ej. HTML de un proxy) se descarta: mensaje estable
 * por status, jamás volcar el body a la UI.
 */
export const toApiError = (error) => {
  if (error instanceof ApiError) return error;

  const status = error?.response?.status ?? 0;
  const envelope = error?.response?.data?.error;
  const isEnvelope = envelope && typeof envelope === 'object' && typeof envelope.message === 'string';

  return new ApiError({
    status,
    code: isEnvelope ? envelope.code : status === 0 ? 'network' : 'unknown',
    message: isEnvelope ? envelope.message : undefined,
    traceId: isEnvelope ? (envelope.trace_id ?? null) : null,
  });
};

/** Cancelaciones de AbortSignal (TanStack Query) — no son errores de UI. */
export const isAbortError = (error) =>
  error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError' || error?.name === 'AbortError';
