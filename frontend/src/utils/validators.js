/**
 * Validation utilities for forms and data
 */

import { VALIDATION } from '../constants';

/**
 * Validate username
 * @param {string} username - Username to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validateUsername = (username) => {
  if (!username || username.trim() === '') {
    return { isValid: false, error: 'Username is required' };
  }
  if (username.length < VALIDATION.MIN_USERNAME_LENGTH) {
    return { 
      isValid: false, 
      error: `Username must be at least ${VALIDATION.MIN_USERNAME_LENGTH} characters` 
    };
  }
  if (!/^[a-zA-Z0-9_]+$/.test(username)) {
    return { 
      isValid: false, 
      error: 'Username can only contain letters, numbers, and underscores' 
    };
  }
  return { isValid: true };
};

/**
 * Validate password
 * @param {string} password - Password to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validatePassword = (password) => {
  if (!password || password.trim() === '') {
    return { isValid: false, error: 'Password is required' };
  }
  if (password.length < VALIDATION.MIN_PASSWORD_LENGTH) {
    return { 
      isValid: false, 
      error: `Password must be at least ${VALIDATION.MIN_PASSWORD_LENGTH} characters` 
    };
  }
  return { isValid: true };
};

/**
 * Validate email format
 * @param {string} email - Email to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validateEmail = (email) => {
  if (!email || email.trim() === '') {
    return { isValid: true }; // Email is optional
  }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(email)) {
    return { isValid: false, error: 'Please enter a valid email address' };
  }
  return { isValid: true };
};

/**
 * Validate phone number format
 * @param {string} phone - Phone number to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validatePhone = (phone) => {
  if (!phone || phone.trim() === '') {
    return { isValid: true }; // Phone is optional
  }
  const phoneRegex = /^[\d\s\-+()]+$/;
  if (!phoneRegex.test(phone)) {
    return { isValid: false, error: 'Please enter a valid phone number' };
  }
  return { isValid: true };
};

/**
 * Validate rating value
 * @param {number} rating - Rating to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validateRating = (rating) => {
  const numRating = Number(rating);
  if (isNaN(numRating)) {
    return { isValid: false, error: 'Rating must be a number' };
  }
  if (numRating < VALIDATION.MIN_RATING || numRating > VALIDATION.MAX_RATING) {
    return { 
      isValid: false, 
      error: `Rating must be between ${VALIDATION.MIN_RATING} and ${VALIDATION.MAX_RATING}` 
    };
  }
  return { isValid: true };
};

/**
 * Validate price value
 * @param {number} price - Price to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validatePrice = (price) => {
  const numPrice = Number(price);
  if (isNaN(numPrice)) {
    return { isValid: false, error: 'Price must be a number' };
  }
  if (numPrice < 0) {
    return { isValid: false, error: 'Price cannot be negative' };
  }
  return { isValid: true };
};

/**
 * Validate date range for reservations
 * @param {string} checkIn - Check-in date
 * @param {string} checkOut - Check-out date
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validateDateRange = (checkIn, checkOut) => {
  if (!checkIn || !checkOut) {
    return { isValid: false, error: 'Both check-in and check-out dates are required' };
  }
  
  const checkInDate = new Date(checkIn);
  const checkOutDate = new Date(checkOut);
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  if (checkInDate < today) {
    return { isValid: false, error: 'Check-in date cannot be in the past' };
  }
  
  if (checkOutDate <= checkInDate) {
    return { isValid: false, error: 'Check-out date must be after check-in date' };
  }

  return { isValid: true };
};

/**
 * Validate URL format
 * @param {string} url - URL to validate
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validateUrl = (url) => {
  if (!url || url.trim() === '') {
    return { isValid: true }; // URL is optional
  }
  try {
    new URL(url);
    return { isValid: true };
  } catch {
    return { isValid: false, error: 'Please enter a valid URL' };
  }
};

/**
 * Validate required field
 * @param {*} value - Value to check
 * @param {string} fieldName - Field name for error message
 * @returns {{ isValid: boolean, error?: string }}
 */
export const validateRequired = (value, fieldName) => {
  if (value === null || value === undefined || value === '') {
    return { isValid: false, error: `${fieldName} is required` };
  }
  return { isValid: true };
};

/**
 * Combine multiple validation results
 * @param {...{ isValid: boolean, error?: string }} validations - Validation results
 * @returns {{ isValid: boolean, errors: string[] }}
 */
export const combineValidations = (...validations) => {
  const errors = validations
    .filter((v) => !v.isValid)
    .map((v) => v.error)
    .filter(Boolean);

  return {
    isValid: errors.length === 0,
    errors,
  };
};
