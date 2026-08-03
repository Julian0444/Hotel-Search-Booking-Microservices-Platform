/**
 * Application constants
 * Centralized configuration values for the application
 */

// API Configuration
// En desarrollo, usa el proxy de Vite (/api/v1) para evitar CORS
// En producción (Docker), usa la URL directa del gateway — https desde el
// plan 10 (cert self-signed en local: aceptar la advertencia del browser)
// La API está versionada bajo /api/v1 (plan 07 / A2)
export const API_CONFIG = {
  BASE_URL: import.meta.env.VITE_API_URL || (import.meta.env.DEV ? '/api/v1' : 'https://localhost/api/v1'),
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

// Reservation status del dominio (hotels-api, plan 04): el backend es el
// dueño del lifecycle; el frontend solo agrupa por fechas para presentación
export const RESERVATION_STATUS = {
  CONFIRMED: 'confirmed',
  CANCELLED: 'cancelled',
};

// Sort options del contrato de search-api (plan 13): whitelist compartida con
// el backend — el orden lo aplica Solr sobre el índice completo
export const SORT_OPTIONS = [
  { value: 'relevance', label: 'Relevance' },
  { value: 'price_asc', label: 'Price: low to high' },
  { value: 'price_desc', label: 'Price: high to low' },
  { value: 'rating_desc', label: 'Top rated' },
];

export const DEFAULT_SORT = 'relevance';

// Catálogo de amenities conocidas (icono + label estable); cualquier otra se
// normaliza con icono genérico — el form admin ofrece estas como opciones
export const AMENITY_CATALOG = [
  'wifi',
  'pool',
  'restaurant',
  'gym',
  'spa',
  'parking',
  'air_conditioning',
  'bar',
  'room_service',
  'laundry',
];

// Límites del booking concierge (el backend valida num_rooms contra el cupo
// real del hotel; esto solo acota los controles de la UI)
export const BOOKING_LIMITS = {
  MAX_ROOMS: 5,
  MAX_GUESTS: 10,
  MAX_STAY_NIGHTS: 30,
};

// Local Storage Keys
export const STORAGE_KEYS = {
  TOKEN: 'token',
  USER: 'user',
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

// Fallback local para hoteles sin imágenes (fase 4: nada de depender de un
// CDN externo para que la UI sea legible)
export const HOTEL_FALLBACK_IMAGE = '/images/hotel-fallback.svg';

// Default Hotel Times ("HH:mm", contrato RV21)
export const DEFAULT_TIMES = {
  CHECK_IN: '15:00',
  CHECK_OUT: '11:00',
};

// Validation
export const VALIDATION = {
  MIN_USERNAME_LENGTH: 3,
  MIN_PASSWORD_LENGTH: 8, // must match users-api RegisterRequest policy
  MAX_RATING: 5,
  MIN_RATING: 0,
};

// Repo público: única promesa "externa" que la Home/Footer pueden mostrar
export const REPO_URL = 'https://github.com/Julian0444/Hotel-Search-Booking-Microservices-Platform';
