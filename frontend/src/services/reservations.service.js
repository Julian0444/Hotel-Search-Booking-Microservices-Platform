/**
 * Reservations Service
 * Handles reservation creation, cancellation, and retrieval
 */

import api from './api';

/**
 * @typedef {import('../types').Reservation} Reservation
 * @typedef {import('../types').ReservationCreateRequest} ReservationCreateRequest
 */

/**
 * Reservations API endpoints
 */
const reservationsService = {
  /**
   * Create a new reservation. The hotel name and total price are derived
   * server-side; num_rooms/num_guests default to 1 when omitted.
   * @param {string} hotelId - Hotel ID
   * @param {string} userId - User ID
   * @param {string} checkIn - Check-in date (YYYY-MM-DD)
   * @param {string} checkOut - Check-out date (YYYY-MM-DD)
   * @param {number} [numRooms=1] - Number of rooms
   * @param {number} [numGuests=1] - Number of guests
   * @returns {Promise<{ id: string }>} Created reservation ID
   */
  create: async (hotelId, userId, checkIn, checkOut, numRooms = 1, numGuests = 1) => {
    const response = await api.post('/reservations', {
      hotel_id: hotelId,
      user_id: userId,
      check_in: checkIn,
      check_out: checkOut,
      num_rooms: numRooms,
      num_guests: numGuests,
    });
    return response.data.data;
  },

  /**
   * Cancel a reservation
   * @param {string} reservationId - Reservation ID
   * @returns {Promise<void>}
   */
  cancel: async (reservationId) => {
    // DELETE exitoso responde 204 sin body (A6)
    await api.delete(`/reservations/${reservationId}`);
  },

  /**
   * Get reservations by user ID
   * @param {string} userId - User ID
   * @returns {Promise<Reservation[]>} List of reservations
   */
  getByUserId: async (userId) => {
    const response = await api.get(`/users/${userId}/reservations`);
    return response.data.data;
  },

  /**
   * Get reservations by user and hotel
   * @param {string} userId - User ID
   * @param {string} hotelId - Hotel ID
   * @returns {Promise<Reservation[]>} List of reservations
   */
  getByUserAndHotel: async (userId, hotelId) => {
    const response = await api.get(`/users/${userId}/hotels/${hotelId}/reservations`);
    return response.data.data;
  },
};

export default reservationsService;
