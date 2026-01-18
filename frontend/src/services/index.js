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
export const healthCheck = async () => {
  const { default: api } = await import('./api');
  const response = await api.get('/health');
  return response.data;
};
