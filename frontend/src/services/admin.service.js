/**
 * Admin Service
 * CRUD de hoteles y observabilidad de la plataforma (hotels-api, admin only).
 */

import api from './api';
import { unwrapObject } from './envelope';

/**
 * @typedef {import('../types').Hotel} Hotel
 * @typedef {import('../types').HotelCreateRequest} HotelCreateRequest
 */

const adminService = {
  /**
   * Create a new hotel
   * @param {HotelCreateRequest} hotelData
   * @returns {Promise<{ id: string }>} Created hotel ID
   */
  createHotel: async (hotelData) => {
    const response = await api.post('/admin/hotels', hotelData);
    return unwrapObject(response.data);
  },

  /**
   * Update an existing hotel
   * @param {string} hotelId
   * @param {HotelCreateRequest} hotelData
   * @returns {Promise<Hotel>} representación actualizada (A6)
   */
  updateHotel: async (hotelId, hotelData) => {
    const response = await api.put(`/admin/hotels/${hotelId}`, hotelData);
    return unwrapObject(response.data);
  },

  /**
   * Delete a hotel
   * @param {string} hotelId
   * @returns {Promise<void>} 204 sin body (A6)
   */
  deleteHotel: async (hotelId) => {
    await api.delete(`/admin/hotels/${hotelId}`);
  },

  /**
   * Estado real de la plataforma (read-only, probes /readyz — plan 11/C2).
   * Los viejos scale/restart/logs eran mocks y ya no existen.
   * @param {{ signal?: AbortSignal }} [options]
   * @returns {Promise<import('../types').MicroservicesStatus>}
   */
  getMicroservicesStatus: async ({ signal } = {}) => {
    const response = await api.get('/admin/microservices', { signal });
    return unwrapObject(response.data);
  },
};

export default adminService;
