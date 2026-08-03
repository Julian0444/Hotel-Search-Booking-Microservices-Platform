/**
 * Admin Service
 * Handles administrative operations for hotels and microservices
 */

import api from './api';

/**
 * @typedef {import('../types').Hotel} Hotel
 * @typedef {import('../types').HotelCreateRequest} HotelCreateRequest
 */

/**
 * Admin API endpoints
 */
const adminService = {
  /**
   * Create a new hotel
   * @param {HotelCreateRequest} hotelData - Hotel data
   * @returns {Promise<{ id: string }>} Created hotel ID
   */
  createHotel: async (hotelData) => {
    const response = await api.post('/admin/hotels', hotelData);
    return response.data.data;
  },

  /**
   * Update an existing hotel
   * @param {string} hotelId - Hotel ID
   * @param {Partial<HotelCreateRequest>} hotelData - Hotel data to update
   * @returns {Promise<void>}
   */
  updateHotel: async (hotelId, hotelData) => {
    // PUT devuelve la representación actualizada dentro de data (A6)
    const response = await api.put(`/admin/hotels/${hotelId}`, hotelData);
    return response.data.data;
  },

  /**
   * Delete a hotel
   * @param {string} hotelId - Hotel ID
   * @returns {Promise<void>}
   */
  deleteHotel: async (hotelId) => {
    // DELETE exitoso responde 204 sin body (A6)
    await api.delete(`/admin/hotels/${hotelId}`);
  },

  /**
   * Get microservices status (read-only, real health via /readyz).
   * The old scale/restart/logs endpoints were mocks and no longer exist.
   * @returns {Promise<{ services: Array, summary: Object }>} Platform status
   */
  getMicroservicesStatus: async () => {
    const response = await api.get('/admin/microservices');
    return response.data.data;
  },
};

export default adminService;
