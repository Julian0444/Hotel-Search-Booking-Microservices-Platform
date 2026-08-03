/**
 * Authentication Service
 * Login/registro y administración de usuarios (users-api).
 */

import api from './api';
import { unwrapList, unwrapObject } from './envelope';

/**
 * @typedef {import('../types').LoginResponse} LoginResponse
 * @typedef {import('../types').User} User
 */

const authService = {
  /**
   * Login user
   * @param {string} username
   * @param {string} password
   * @returns {Promise<LoginResponse>} user_id llega como string (A7)
   */
  login: async (username, password) => {
    const response = await api.post('/login', { username, password });
    return unwrapObject(response.data);
  },

  /**
   * Register new user (role is always assigned server-side as "cliente")
   * @param {string} username
   * @param {string} password
   * @returns {Promise<{ id: string }>} Created user ID
   */
  register: async (username, password) => {
    const response = await api.post('/users', { username, password });
    return unwrapObject(response.data);
  },

  /**
   * Lista paginada de usuarios (admin only).
   * @param {{ limit?: number, offset?: number }} [params]
   * @param {{ signal?: AbortSignal }} [options]
   * @returns {Promise<import('../types').ListPage<User>>}
   */
  listUsers: async ({ limit = 20, offset = 0 } = {}, { signal } = {}) => {
    const response = await api.get(`/users?limit=${limit}&offset=${offset}`, { signal });
    return unwrapList(response.data);
  },

  /**
   * Delete user (owner or admin)
   * @param {string} id - User ID (string, A7)
   * @returns {Promise<void>} 204 sin body (A6)
   */
  deleteUser: async (id) => {
    await api.delete(`/users/${id}`);
  },
};

export default authService;
