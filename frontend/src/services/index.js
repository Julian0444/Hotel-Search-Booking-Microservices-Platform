/**
 * Services barrel export
 * Centralized exports for all API services
 */

export { default as api } from './api';
export { default as authService } from './auth.service';
export { default as hotelsService } from './hotels.service';
export { default as reservationsService } from './reservations.service';
export { default as adminService } from './admin.service';

// Health check utility
// El /health del gateway NO está versionado (A2): se pega relativo al host,
// no al BASE_URL /api/v1 de la instancia de axios.
export const healthCheck = async () => {
  const { default: axios } = await import('axios');
  const response = await axios.get('/health');
  return response.data;
};
