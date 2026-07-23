/**
 * Authentication Service
 * Handles user authentication, registration, and user management
 */

import api from './api';

/**
 * @typedef {import('../types').LoginRequest} LoginRequest
 * @typedef {import('../types').LoginResponse} LoginResponse
 * @typedef {import('../types').RegisterRequest} RegisterRequest
 * @typedef {import('../types').User} User
 */

/**
 * Authentication API endpoints
 */
const authService = {
  /**
   * Login user
   * @param {string} username - Username
   * @param {string} password - Password
   * @returns {Promise<LoginResponse>} Login response with token
   */
  login: async (username, password) => {
    const response = await api.post('/login', { username, password });
    return response.data.data;
  },

  /**
   * Register new user (role is always assigned server-side as "cliente")
   * @param {string} username - Username
   * @param {string} password - Password
   * @returns {Promise<{ id: string }>} Created user ID
   */
  register: async (username, password) => {
    const response = await api.post('/users', { username, password });
    return response.data.data;
  },

  /**
   * Get all users (admin only)
   * @returns {Promise<User[]>} List of users
   */
  getAllUsers: async () => {
    const response = await api.get('/users');
    return response.data.data;
  },

  /**
   * Get user by ID
   * @param {number} id - User ID
   * @returns {Promise<User>} User data
   */
  getUserById: async (id) => {
    const response = await api.get(`/users/${id}`);
    return response.data.data;
  },

  /**
   * Delete user (admin only)
   * @param {number} id - User ID
   * @returns {Promise<void>}
   */
  deleteUser: async (id) => {
    // DELETE exitoso responde 204 sin body (A6)
    await api.delete(`/users/${id}`);
  },
};

export default authService;
