/**
 * Validation utilities (plan 13, F13-08): predicados puros que usan las
 * reglas de react-hook-form del HotelForm y los tests.
 */

/**
 * URL de imagen válida: http(s) absoluta (el backend no valida esto; la UI
 * no debe aceptar javascript:, data: ni paths relativos).
 * @param {string} value
 * @returns {boolean}
 */
export const isHttpUrl = (value) => {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
};

/**
 * Hora del día "HH:mm" del contrato (RV21) — espejo de validTimeOfDay de
 * hotels-api.
 * @param {string} value
 * @returns {boolean}
 */
export const isTimeOfDay = (value) =>
  /^([01]\d|2[0-3]):[0-5]\d$/.test(value || '');

/**
 * Email con shape razonable (opcional en el contrato: vacío es válido).
 * @param {string} value
 * @returns {boolean}
 */
export const isOptionalEmail = (value) =>
  !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
