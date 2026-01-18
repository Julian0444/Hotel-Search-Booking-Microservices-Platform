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
    return response.data;
  },

  /**
   * Update an existing hotel
   * @param {string} hotelId - Hotel ID
   * @param {Partial<HotelCreateRequest>} hotelData - Hotel data to update
   * @returns {Promise<void>}
   */
  updateHotel: async (hotelId, hotelData) => {
    const response = await api.put(`/admin/hotels/${hotelId}`, hotelData);
    return response.data;
  },

  /**
   * Delete a hotel
   * @param {string} hotelId - Hotel ID
   * @returns {Promise<void>}
   */
  deleteHotel: async (hotelId) => {
    const response = await api.delete(`/admin/hotels/${hotelId}`);
    return response.data;
  },

  /**
   * Get microservices status
   * @returns {Promise<Object>} Microservices status
   */
  getMicroservicesStatus: async () => {
    const response = await api.get('/admin/microservices');
    return response.data;
  },

  /**
   * Scale a service
   * @param {string} serviceName - Service name
   * @param {number} replicas - Number of replicas
   * @returns {Promise<void>}
   */
  scaleService: async (serviceName, replicas) => {
    const response = await api.post('/admin/microservices/scale', {
      service_name: serviceName,
      replicas,
    });
    return response.data;
  },

  /**
   * Get service logs
   * @param {string} serviceName - Service name
   * @returns {Promise<{ logs: string[] }>} Service logs
   */
  getServiceLogs: async (serviceName) => {
    const response = await api.get(`/admin/microservices/${serviceName}/logs`);
    return response.data;
  },

  /**
   * Restart a service
   * @param {string} serviceName - Service name
   * @returns {Promise<void>}
   */
  restartService: async (serviceName) => {
    const response = await api.post(`/admin/microservices/${serviceName}/restart`);
    return response.data;
  },
};

export default adminService;
