/**
 * Hotels Service
 * Búsqueda (Solr vía search-api), catálogo y disponibilidad (hotels-api).
 * Devuelve SIEMPRE el shape final del contrato (plan 13, F13-01): nada de
 * camelCase ni fallbacks legacy aguas arriba de este archivo.
 */

import api from './api';
import { unwrapList, unwrapObject } from './envelope';
import { PAGINATION } from '../constants';

/**
 * @typedef {import('../types').Hotel} Hotel
 */

const hotelsService = {
  /**
   * Search hotels (global, ordenado por Solr — plan 13).
   * @param {{ q?: string, offset?: number, limit?: number, sort?: string }} params
   * @param {{ signal?: AbortSignal }} [options] - AbortSignal de TanStack Query
   * @returns {Promise<import('../types').ListPage<Hotel>>}
   */
  search: async (
    { q = '', offset = PAGINATION.DEFAULT_OFFSET, limit = PAGINATION.DEFAULT_PAGE_SIZE, sort = '' } = {},
    { signal } = {},
  ) => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    params.set('offset', String(offset));
    params.set('limit', String(limit));
    if (sort && sort !== 'relevance') params.set('sort', sort);

    const response = await api.get(`/search?${params.toString()}`, { signal });
    return unwrapList(response.data);
  },

  /**
   * Catálogo paginado desde hotels-api (fuente de verdad, sin lag de índice).
   * @param {{ offset?: number, limit?: number }} params
   * @param {{ signal?: AbortSignal }} [options]
   * @returns {Promise<import('../types').ListPage<Hotel>>}
   */
  list: async ({ offset = 0, limit = PAGINATION.DEFAULT_PAGE_SIZE } = {}, { signal } = {}) => {
    const response = await api.get(`/hotels?limit=${limit}&offset=${offset}`, { signal });
    return unwrapList(response.data);
  },

  /**
   * Get hotel by ID
   * @param {string} hotelId - Hotel ID
   * @param {{ signal?: AbortSignal }} [options]
   * @returns {Promise<Hotel>} Hotel details
   */
  getById: async (hotelId, { signal } = {}) => {
    const response = await api.get(`/hotels/${hotelId}`, { signal });
    return unwrapObject(response.data);
  },

  /**
   * Disponibilidad para un rango (feedback rápido; POST /reservations es la
   * autoridad atómica).
   * @param {string[]} hotelIds
   * @param {string} checkIn - "YYYY-MM-DD"
   * @param {string} checkOut - "YYYY-MM-DD"
   * @param {{ signal?: AbortSignal }} [options]
   * @returns {Promise<Object.<string, boolean>>} hotelId → disponible
   */
  checkAvailability: async (hotelIds, checkIn, checkOut, { signal } = {}) => {
    const response = await api.post(
      '/hotels/availability',
      { hotel_ids: hotelIds, check_in: checkIn, check_out: checkOut },
      { signal },
    );
    return unwrapObject(response.data);
  },
};

export default hotelsService;
