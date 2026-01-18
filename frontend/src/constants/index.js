/**
 * Application constants
 * Centralized configuration values for the application
 */

// API Configuration
// En desarrollo, usa el proxy de Vite (/api) para evitar CORS
// En producción (Docker), usa la URL directa
export const API_CONFIG = {
  BASE_URL: import.meta.env.VITE_API_URL || (import.meta.env.DEV ? '/api' : 'http://localhost'),
  TIMEOUT: 30000,
};

// Pagination
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 12,
  DEFAULT_OFFSET: 0,
};

// User Roles
export const USER_ROLES = {
  ADMIN: 'administrador',
  CLIENT: 'cliente',
};

// Reservation Status
export const RESERVATION_STATUS = {
  UPCOMING: 'upcoming',
  IN_PROGRESS: 'in_progress',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
};

// Hotel Amenities
export const AMENITIES = {
  WIFI: 'wifi',
  POOL: 'pool',
  RESTAURANT: 'restaurant',
  GYM: 'gym',
  SPA: 'spa',
  PARKING: 'parking',
  AC: 'air_conditioning',
  BAR: 'bar',
  ROOM_SERVICE: 'room_service',
  LAUNDRY: 'laundry',
};

// Sort Options
export const SORT_OPTIONS = {
  RELEVANCE: 'relevance',
  PRICE_LOW: 'price-low',
  PRICE_HIGH: 'price-high',
  RATING: 'rating',
};

// Local Storage Keys
export const STORAGE_KEYS = {
  TOKEN: 'token',
  USER: 'user',
  THEME: 'theme',
};

// Routes
export const ROUTES = {
  HOME: '/',
  SEARCH: '/search',
  LOGIN: '/login',
  REGISTER: '/register',
  HOTEL_DETAIL: '/hotels/:id',
  RESERVATIONS: '/reservations',
  ADMIN: '/admin',
  ADMIN_NEW_HOTEL: '/admin/hotels/new',
  ADMIN_EDIT_HOTEL: '/admin/hotels/:id/edit',
};

// Placeholder Images
export const PLACEHOLDER_IMAGES = [
  'https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800',
  'https://images.unsplash.com/photo-1582719508461-905c673771fd?w=800',
  'https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800',
  'https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?w=800',
  'https://images.unsplash.com/photo-1571896349842-33c89424de2d?w=800',
  'https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800',
];

// Default Hotel Times
export const DEFAULT_TIMES = {
  CHECK_IN: '15:00',
  CHECK_OUT: '11:00',
};

// Validation
export const VALIDATION = {
  MIN_USERNAME_LENGTH: 3,
  MIN_PASSWORD_LENGTH: 6,
  MAX_RATING: 5,
  MIN_RATING: 0,
};
