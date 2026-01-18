/**
 * Utility helper functions
 */

import { format, parseISO, isPast, isToday, isFuture } from 'date-fns';
import { PLACEHOLDER_IMAGES } from '../constants';

/**
 * Get a placeholder image based on hotel ID
 * @param {string} hotelId - Hotel ID
 * @param {string[]} [hotelImages] - Hotel's own images
 * @returns {string} Image URL
 */
export const getHotelImage = (hotelId, hotelImages = []) => {
  if (hotelImages && hotelImages.length > 0) {
    return hotelImages[0];
  }
  const index = parseInt(hotelId?.slice(-2) || '0', 16) % PLACEHOLDER_IMAGES.length;
  return PLACEHOLDER_IMAGES[index];
};

/**
 * Format a date string to a readable format
 * @param {string} dateString - ISO date string
 * @param {string} [formatStr='MMMM d, yyyy'] - date-fns format string
 * @returns {string} Formatted date
 */
export const formatDate = (dateString, formatStr = 'MMMM d, yyyy') => {
  try {
    return format(parseISO(dateString), formatStr);
  } catch {
    return dateString;
  }
};

/**
 * Calculate number of nights between two dates
 * @param {string} checkIn - Check-in date
 * @param {string} checkOut - Check-out date
 * @returns {number} Number of nights
 */
export const calculateNights = (checkIn, checkOut) => {
  try {
    const start = parseISO(checkIn);
    const end = parseISO(checkOut);
    return Math.ceil((end - start) / (1000 * 60 * 60 * 24));
  } catch {
    return 0;
  }
};

/**
 * Calculate total price for a stay
 * @param {number} pricePerNight - Price per night
 * @param {string} checkIn - Check-in date
 * @param {string} checkOut - Check-out date
 * @returns {number} Total price
 */
export const calculateTotalPrice = (pricePerNight, checkIn, checkOut) => {
  const nights = calculateNights(checkIn, checkOut);
  return pricePerNight * nights;
};

/**
 * Get reservation status based on dates
 * @param {string} checkIn - Check-in date
 * @param {string} checkOut - Check-out date
 * @returns {{ label: string, color: string, canCancel: boolean }}
 */
export const getReservationStatus = (checkIn, checkOut) => {
  const checkInDate = parseISO(checkIn);
  const checkOutDate = parseISO(checkOut);

  if (isPast(checkOutDate) && !isToday(checkOutDate)) {
    return { label: 'Completed', color: 'default', canCancel: false };
  }
  if (isToday(checkInDate) || (isPast(checkInDate) && isFuture(checkOutDate))) {
    return { label: 'In Progress', color: 'success', canCancel: false };
  }
  if (isFuture(checkInDate)) {
    return { label: 'Upcoming', color: 'primary', canCancel: true };
  }
  return { label: 'Pending', color: 'warning', canCancel: true };
};

/**
 * Get minimum check-in date (today)
 * @returns {string} ISO date string
 */
export const getMinCheckInDate = () => {
  return new Date().toISOString().split('T')[0];
};

/**
 * Get minimum check-out date (day after check-in)
 * @param {string} checkIn - Check-in date
 * @returns {string} ISO date string
 */
export const getMinCheckOutDate = (checkIn) => {
  if (!checkIn) return getMinCheckInDate();
  const nextDay = new Date(checkIn);
  nextDay.setDate(nextDay.getDate() + 1);
  return nextDay.toISOString().split('T')[0];
};

/**
 * Format price with currency
 * @param {number} price - Price value
 * @param {string} [currency='USD'] - Currency code
 * @returns {string} Formatted price
 */
export const formatPrice = (price, currency = 'USD') => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(price);
};

/**
 * Truncate text to a specified length
 * @param {string} text - Text to truncate
 * @param {number} [maxLength=100] - Maximum length
 * @returns {string} Truncated text
 */
export const truncateText = (text, maxLength = 100) => {
  if (!text || text.length <= maxLength) return text;
  return `${text.substring(0, maxLength)}...`;
};

/**
 * Debounce function execution
 * @param {Function} func - Function to debounce
 * @param {number} [wait=300] - Wait time in ms
 * @returns {Function} Debounced function
 */
export const debounce = (func, wait = 300) => {
  let timeout;
  return (...args) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(this, args), wait);
  };
};

/**
 * Get initials from a username
 * @param {string} username - Username
 * @returns {string} Initials (1-2 characters)
 */
export const getInitials = (username) => {
  if (!username) return '?';
  return username.charAt(0).toUpperCase();
};

/**
 * Check if a value is empty (null, undefined, empty string, empty array)
 * @param {*} value - Value to check
 * @returns {boolean} True if empty
 */
export const isEmpty = (value) => {
  if (value === null || value === undefined) return true;
  if (typeof value === 'string') return value.trim() === '';
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === 'object') return Object.keys(value).length === 0;
  return false;
};
