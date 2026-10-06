/**
 * Envelope del contrato /api/v1 (A5): objetos en {data}, listas en
 * {data, meta:{total,limit,offset}}. Los componentes NUNCA leen el envelope:
 * services/hooks normalizan acá una sola vez (plan 13, F13-01).
 */

/** {data: T} → T. Rechaza shapes inesperados en vez de propagar undefined. */
export const unwrapObject = (body) => {
  if (!body || typeof body !== 'object' || !('data' in body)) {
    throw new Error('unexpected API response: missing data envelope');
  }
  return body.data;
};

/**
 * {data: T[], meta} → {items, total, limit, offset}.
 * meta.total puede faltar en algunos listados (reservas de hotels-api):
 * se degrada al largo de la página, nunca a undefined.
 */
export const unwrapList = (body) => {
  if (!body || typeof body !== 'object' || !Array.isArray(body.data)) {
    throw new Error('unexpected API response: missing list envelope');
  }
  const meta = body.meta && typeof body.meta === 'object' ? body.meta : {};
  return {
    items: body.data,
    total: typeof meta.total === 'number' ? meta.total : body.data.length,
    limit: typeof meta.limit === 'number' ? meta.limit : body.data.length,
    offset: typeof meta.offset === 'number' ? meta.offset : 0,
  };
};
