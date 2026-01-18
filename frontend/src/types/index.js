/**
 * Type definitions using JSDoc for better IDE support and documentation
 * These types help with validation, autocompletion, and serve as documentation
 */

/**
 * @typedef {Object} User
 * @property {number} id - User ID
 * @property {string} username - Username
 * @property {('cliente'|'administrador')} tipo - User role
 */

/**
 * @typedef {Object} LoginRequest
 * @property {string} username - Username for login
 * @property {string} password - Password for login
 */

/**
 * @typedef {Object} RegisterRequest
 * @property {string} username - Username for registration
 * @property {string} password - Password for registration
 * @property {('cliente'|'administrador')} [tipo='cliente'] - User role
 */

/**
 * @typedef {Object} LoginResponse
 * @property {number} user_id - User ID
 * @property {string} username - Username
 * @property {string} token - JWT token
 * @property {string} tipo - User role
 */

/**
 * @typedef {Object} Hotel
 * @property {string} id - Hotel ID (MongoDB ObjectId)
 * @property {string} name - Hotel name
 * @property {string} description - Hotel description
 * @property {string} address - Street address
 * @property {string} city - City
 * @property {string} state - State/Province
 * @property {string} country - Country
 * @property {string} phone - Contact phone
 * @property {string} email - Contact email
 * @property {number} price_per_night - Price per night in USD
 * @property {number} rating - Rating (0-5)
 * @property {number} avaiable_rooms - Available rooms count
 * @property {string} check_in_time - Check-in time (HH:mm)
 * @property {string} check_out_time - Check-out time (HH:mm)
 * @property {string[]} amenities - List of amenities
 * @property {string[]} images - List of image URLs
 */

/**
 * @typedef {Object} HotelCreateRequest
 * @property {string} name - Hotel name
 * @property {string} description - Hotel description
 * @property {string} address - Street address
 * @property {string} city - City
 * @property {string} [state] - State/Province
 * @property {string} country - Country
 * @property {string} [phone] - Contact phone
 * @property {string} [email] - Contact email
 * @property {number} price_per_night - Price per night
 * @property {number} [rating=0] - Initial rating
 * @property {number} avaiable_rooms - Available rooms
 * @property {string} [check_in_time='15:00'] - Check-in time
 * @property {string} [check_out_time='11:00'] - Check-out time
 * @property {string[]} [amenities=[]] - Amenities list
 * @property {string[]} [images=[]] - Image URLs
 */

/**
 * @typedef {Object} Reservation
 * @property {string} id - Reservation ID
 * @property {string} hotel_id - Hotel ID
 * @property {string} hotel_name - Hotel name
 * @property {number} user_id - User ID
 * @property {string} check_in - Check-in date (YYYY-MM-DD)
 * @property {string} check_out - Check-out date (YYYY-MM-DD)
 */

/**
 * @typedef {Object} ReservationCreateRequest
 * @property {string} hotel_id - Hotel ID
 * @property {string} hotel_name - Hotel name
 * @property {string} check_in - Check-in date
 * @property {string} check_out - Check-out date
 */

/**
 * @typedef {Object} AvailabilityRequest
 * @property {string[]} hotel_ids - List of hotel IDs to check
 * @property {string} check_in - Check-in date
 * @property {string} check_out - Check-out date
 */

/**
 * @typedef {Object.<string, boolean>} AvailabilityResponse
 * Map of hotel ID to availability status
 */

/**
 * @typedef {Object} SearchParams
 * @property {string} [q=''] - Search query
 * @property {number} [offset=0] - Pagination offset
 * @property {number} [limit=20] - Results limit
 */

/**
 * @typedef {Object} ApiError
 * @property {string} error - Error message
 * @property {number} [status] - HTTP status code
 */

/**
 * @typedef {Object} PaginatedResponse
 * @template T
 * @property {T[]} data - Array of items
 * @property {number} total - Total count
 * @property {number} offset - Current offset
 * @property {number} limit - Page size
 */

// Export empty object for module resolution
export default {};
