/**
 * Reservations Service
 * Alta idempotente, cancelación e historial (hotels-api).
 */

import api from './api';
import { unwrapList, unwrapObject } from './envelope';

/**
 * @typedef {import('../types').Reservation} Reservation
 */

const reservationsService = {
  /**
   * Crea una reserva. user_id sale del JWT server-side; hotel_name y
   * total_price se derivan en hotels-api. La Idempotency-Key viaja por
   * header (A3): mismo key+user → misma respuesta, sin duplicar (F13-05).
   * @param {import('../types').ReservationCreateRequest} request
   * @param {{ idempotencyKey?: string }} [options]
   * @returns {Promise<{ id: string }>}
   */
  create: async ({ hotel_id, check_in, check_out, num_rooms = 1, num_guests = 1 }, { idempotencyKey } = {}) => {
    const response = await api.post(
      '/reservations',
      { hotel_id, check_in, check_out, num_rooms, num_guests },
      idempotencyKey ? { headers: { 'Idempotency-Key': idempotencyKey } } : undefined,
    );
    return unwrapObject(response.data);
  },

  /**
   * Cancela una reserva (soft-delete server-side; queda en el historial).
   * @param {string} reservationId
   * @returns {Promise<void>} 204 sin body (A6)
   */
  cancel: async (reservationId) => {
    await api.delete(`/reservations/${reservationId}`);
  },

  /**
   * Historial del usuario (canceladas incluidas — F13-06).
   * El meta de este endpoint no trae total (hotels-api): total degrada al
   * largo de la página.
   * @param {string} userId
   * @param {{ limit?: number, offset?: number }} [params]
   * @param {{ signal?: AbortSignal }} [options]
   * @returns {Promise<import('../types').ListPage<Reservation>>}
   */
  listByUser: async (userId, { limit = 100, offset = 0 } = {}, { signal } = {}) => {
    const response = await api.get(`/users/${userId}/reservations?limit=${limit}&offset=${offset}`, { signal });
    return unwrapList(response.data);
  },
};

export default reservationsService;
