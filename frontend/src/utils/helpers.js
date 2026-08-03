/**
 * Utility helper functions (plan 13: lo de fechas vive en dateOnly.js y lo
 * de dinero en money.js; acá queda solo presentación genérica).
 */

import { HOTEL_FALLBACK_IMAGE } from '../constants';

/**
 * Primera imagen del hotel o el fallback LOCAL (nunca un CDN externo).
 * @param {import('../types').Hotel} hotel
 * @returns {string} Image URL
 */
export const hotelImage = (hotel) =>
  hotel?.images?.length ? hotel.images[0] : HOTEL_FALLBACK_IMAGE;

/**
 * Truncate text to a specified length
 * @param {string} text - Text to truncate
 * @param {number} [maxLength=100] - Maximum length
 * @returns {string} Truncated text
 */
export const truncateText = (text, maxLength = 100) => {
  if (!text || text.length <= maxLength) return text;
  return `${text.substring(0, maxLength)}…`;
};

/**
 * Get initials from a username
 * @param {string} username - Username
 * @returns {string} Initials (1 character)
 */
export const getInitials = (username) => {
  if (!username) return '?';
  return username.charAt(0).toUpperCase();
};

/**
 * Etiqueta legible de una amenity del catálogo o libre:
 * "air_conditioning" → "Air conditioning".
 * @param {string} amenity
 * @returns {string}
 */
export const amenityLabel = (amenity) => {
  const clean = (amenity || '').replace(/_/g, ' ').trim();
  return clean ? clean.charAt(0).toUpperCase() + clean.slice(1) : '';
};

/**
 * ID corto para mostrar/copiar (reservas): últimos 8 caracteres.
 * @param {string} id
 * @returns {string}
 */
export const shortId = (id) => (id || '').slice(-8).toUpperCase();
