/**
 * Type definitions using JSDoc for better IDE support and documentation
 * These mirror the FINAL wire contract (plans 07/11): snake_case, string
 * user ids, "HH:mm" times, civil "YYYY-MM-DD" dates and cents for money.
 */

/**
 * @typedef {Object} User
 * @property {string} id - User ID (serialized as string across the platform, A7)
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
 * @property {string} username - Username for registration (3-50 chars)
 * @property {string} password - Password for registration (8-72 chars)
 * Role is always assigned server-side as "cliente"; admins are seeded via env.
 */

/**
 * @typedef {Object} LoginResponse
 * @property {string} user_id - User ID (string, A7)
 * @property {string} username - Username
 * @property {string} token - JWT token (HS256; claims user_id/username/tipo/iss/aud/exp)
 * @property {('cliente'|'administrador')} tipo - User role
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
 * @property {number} price_per_night - Price per night in major units (float64)
 * @property {number} rating - Rating (0-5)
 * @property {number} available_rooms - Available rooms count
 * @property {string} check_in_time - Check-in time ("HH:mm")
 * @property {string} check_out_time - Check-out time ("HH:mm")
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
 * @property {number} price_per_night - Price per night in major units
 * @property {number} [rating=0] - Initial rating
 * @property {number} available_rooms - Available rooms
 * @property {string} [check_in_time='15:00'] - Check-in time ("HH:mm")
 * @property {string} [check_out_time='11:00'] - Check-out time ("HH:mm")
 * @property {string[]} [amenities=[]] - Amenities list
 * @property {string[]} [images=[]] - Image URLs
 */

/**
 * @typedef {Object} Reservation
 * @property {string} id - Reservation ID
 * @property {string} hotel_id - Hotel ID
 * @property {string} hotel_name - Hotel name (derived server-side)
 * @property {string} user_id - User ID (string, A7)
 * @property {string} check_in - Check-in date ("YYYY-MM-DD", civil)
 * @property {string} check_out - Check-out date ("YYYY-MM-DD", civil)
 * @property {('confirmed'|'cancelled')} status - Lifecycle owned by the backend
 * @property {number} num_rooms - Rooms reserved
 * @property {number} num_guests - Guests
 * @property {number} total_price - Total in CENTS (int64)
 * @property {string} currency - ISO currency code (e.g. "USD")
 * @property {string} created_at - RFC3339 audit timestamp
 * @property {string} [cancelled_at] - RFC3339, present when cancelled
 */

/**
 * @typedef {Object} ReservationCreateRequest
 * @property {string} hotel_id - Hotel ID
 * @property {string} check_in - Check-in date ("YYYY-MM-DD")
 * @property {string} check_out - Check-out date ("YYYY-MM-DD")
 * @property {number} [num_rooms=1] - Rooms (1..available)
 * @property {number} [num_guests=1] - Guests
 * user_id is taken from the JWT server-side; hotel_name is derived.
 */

/**
 * @typedef {Object} AvailabilityRequest
 * @property {string[]} hotel_ids - List of hotel IDs to check
 * @property {string} check_in - Check-in date ("YYYY-MM-DD")
 * @property {string} check_out - Check-out date ("YYYY-MM-DD")
 */

/**
 * @typedef {Object.<string, boolean>} AvailabilityResponse
 * Map of hotel ID to availability status
 */

/**
 * @typedef {Object} ListPage
 * @template T
 * @property {T[]} items - Page of items (envelope data)
 * @property {number} total - Real total from meta (RV22)
 * @property {number} limit - Page size echoed by the API
 * @property {number} offset - Offset echoed by the API
 */

/**
 * @typedef {Object} ServiceInstanceStatus
 * @property {string} name
 * @property {string} url
 * @property {('up'|'down')} status
 * @property {number} latency_ms
 * @property {string} [error]
 */

/**
 * @typedef {Object} ServiceStatus
 * @property {string} name
 * @property {('up'|'degraded'|'down')} status
 * @property {boolean} load_balanced
 * @property {ServiceInstanceStatus[]} instances
 */

/**
 * @typedef {Object} MicroservicesStatus
 * @property {ServiceStatus[]} services
 * @property {{total_services: number, total_instances: number, healthy_services: number}} summary
 */

// Export empty object for module resolution
export default {};
