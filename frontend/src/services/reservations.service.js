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
   * Create a new reservation
   * @param {string} hotelId - Hotel ID
   * @param {string} hotelName - Hotel name
   * @param {string} checkIn - Check-in date
   * @param {string} checkOut - Check-out date
   * @returns {Promise<{ id: string }>} Created reservation ID
   */
  create: async (hotelId, hotelName, checkIn, checkOut) => {
    const response = await api.post('/reservations', {
      hotel_id: hotelId,
      hotel_name: hotelName,
      check_in: checkIn,
      check_out: checkOut,
    });
    return response.data;
  },

  /**
   * Cancel a reservation
   * @param {string} reservationId - Reservation ID
   * @returns {Promise<void>}
   */
  cancel: async (reservationId) => {
    const response = await api.delete(`/reservations/${reservationId}`);
    return response.data;
  },

  /**
   * Get reservations by user ID
   * @param {number} userId - User ID
   * @returns {Promise<Reservation[]>} List of reservations
   */
  getByUserId: async (userId) => {
    const response = await api.get(`/users/${userId}/reservations`);
    return response.data;
  },

  /**
   * Get reservations by user and hotel
   * @param {number} userId - User ID
   * @param {string} hotelId - Hotel ID
   * @returns {Promise<Reservation[]>} List of reservations
   */
  getByUserAndHotel: async (userId, hotelId) => {
    const response = await api.get(`/users/${userId}/hotels/${hotelId}/reservations`);
    return response.data;
  },
};

export default reservationsService;
