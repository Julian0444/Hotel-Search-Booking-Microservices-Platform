/**
 * Hotels Service
 * Handles hotel search, details, and availability checking
 */

import api from './api';
import { PAGINATION } from '../constants';

/**
 * @typedef {import('../types').Hotel} Hotel
 * @typedef {import('../types').SearchParams} SearchParams
 * @typedef {import('../types').AvailabilityRequest} AvailabilityRequest
 */

/**
 * Hotels API endpoints
 */
const hotelsService = {
  /**
   * Search hotels
   * @param {string} [query=''] - Search query
   * @param {number} [offset=0] - Pagination offset
   * @param {number} [limit=20] - Results limit
   * @returns {Promise<Hotel[]>} List of hotels
   */
  search: async (query = '', offset = PAGINATION.DEFAULT_OFFSET, limit = PAGINATION.DEFAULT_PAGE_SIZE) => {
    const params = new URLSearchParams();
    if (query) params.append('q', query);
    params.append('offset', offset.toString());
    params.append('limit', limit.toString());

    const response = await api.get(`/search?${params.toString()}`);
    return response.data;
  },

  /**
   * Get hotel by ID
   * @param {string} hotelId - Hotel ID
   * @returns {Promise<Hotel>} Hotel details
   */
  getById: async (hotelId) => {
    const response = await api.get(`/hotels/${hotelId}`);
    return response.data;
  },

  /**
   * Get reservations for a hotel
   * @param {string} hotelId - Hotel ID
   * @returns {Promise<import('../types').Reservation[]>} List of reservations
   */
  getReservationsByHotel: async (hotelId) => {
    const response = await api.get(`/hotels/${hotelId}/reservations`);
    return response.data;
  },

  /**
   * Check availability for hotels
   * @param {string[]} hotelIds - List of hotel IDs
   * @param {string} checkIn - Check-in date
   * @param {string} checkOut - Check-out date
   * @returns {Promise<Object.<string, boolean>>} Availability map
   */
  checkAvailability: async (hotelIds, checkIn, checkOut) => {
    const response = await api.post('/hotels/availability', {
      hotel_ids: hotelIds,
      check_in: checkIn,
      check_out: checkOut,
    });
    return response.data;
  },
};

export default hotelsService;
