/**
 * Fechas civiles "YYYY-MM-DD" (plan 13, F13-04).
 *
 * Regla de la casa: una fecha de reserva NO es un instante — es un string
 * civil. Se construye por componentes LOCALES (lo que el usuario ve en su
 * calendario) y se opera en aritmética UTC pura (inmune a DST). Prohibido
 * derivar el día recortando el ISO string UTC (corre el día al oeste de UTC
 * cerca de medianoche) y `new Date('YYYY-MM-DD')` en lógica de producto
 * (parsea UTC y "ayer" aparece en getDate() local).
 */

const DATE_ONLY_RE = /^(\d{4})-(\d{2})-(\d{2})$/;

/** @returns {boolean} true si es un "YYYY-MM-DD" calendario válido */
export const isValidDateOnly = (value) => {
  const match = DATE_ONLY_RE.exec(value || '');
  if (!match) return false;
  const [, y, m, d] = match.map(Number);
  const probe = new Date(Date.UTC(y, m - 1, d));
  return probe.getUTCFullYear() === y && probe.getUTCMonth() === m - 1 && probe.getUTCDate() === d;
};

const pad = (n) => String(n).padStart(2, '0');

const fromParts = (y, m, d) => `${y}-${pad(m)}-${pad(d)}`;

/** Milisegundos UTC de la medianoche civil (solo para aritmética interna). */
const utcMillis = (dateOnly) => {
  const [, y, m, d] = DATE_ONLY_RE.exec(dateOnly).map(Number);
  return Date.UTC(y, m - 1, d);
};

/** Hoy según el calendario LOCAL del usuario. */
export const todayLocal = (now = new Date()) =>
  fromParts(now.getFullYear(), now.getMonth() + 1, now.getDate());

/** Suma días calendario sin pasar por horas locales (DST-safe). */
export const addDaysDateOnly = (dateOnly, days) => {
  const date = new Date(utcMillis(dateOnly) + days * 86_400_000);
  return fromParts(date.getUTCFullYear(), date.getUTCMonth() + 1, date.getUTCDate());
};

/** Noches entre dos fechas civiles; entero exacto aunque cruce DST. */
export const differenceInNights = (checkIn, checkOut) =>
  Math.round((utcMillis(checkOut) - utcMillis(checkIn)) / 86_400_000);

/** -1 | 0 | 1 comparando fechas civiles (orden lexicográfico = cronológico). */
export const compareDateOnly = (a, b) => (a < b ? -1 : a > b ? 1 : 0);

/**
 * Etiqueta legible de una fecha civil. Se formatea anclada a UTC para que la
 * etiqueta muestre EXACTAMENTE el día del string en cualquier huso.
 */
export const formatDateOnlyLabel = (dateOnly, options = {}) => {
  if (!isValidDateOnly(dateOnly)) return dateOnly || '';
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    ...options,
    timeZone: 'UTC',
  }).format(new Date(utcMillis(dateOnly)));
};

/**
 * Etiqueta legible de una hora del día "HH:mm" del contrato (RV21).
 * Nunca RFC3339: "15:00" → "3:00 PM".
 */
export const formatTimeLabel = (timeOfDay) => {
  const match = /^(\d{2}):(\d{2})$/.exec(timeOfDay || '');
  if (!match) return timeOfDay || '';
  return new Intl.DateTimeFormat('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    timeZone: 'UTC',
  }).format(new Date(Date.UTC(2000, 0, 1, Number(match[1]), Number(match[2]))));
};
