/**
 * Dinero (plan 13, F13-04): el dominio maneja DOS unidades y no se mezclan.
 * - hotel.price_per_night viaja en unidades mayores (float64);
 * - reservation.total_price viaja en CENTAVOS (int64).
 * El preview del booking se calcula en centavos espejando el backend
 * (hotels-api: round(price*100) * nights * rooms) para que el total mostrado
 * coincida al centavo con el persistido.
 */

/** Unidades mayores (price_per_night): "$150.50", "$320". */
export const formatMajorAmount = (amount, currency = 'USD') => {
  const value = amount ?? 0;
  const digits = Number.isInteger(value) ? 0 : 2;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value);
};

/** Centavos (reservation.total_price): 45150 → "$451.50". */
export const formatMinorAmount = (cents, currency = 'USD') =>
  new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format((cents ?? 0) / 100);

/** Total estimado en centavos, espejo exacto del cálculo de hotels-api. */
export const previewTotalCents = (nightlyRate, nights, numRooms) =>
  Math.round((nightlyRate ?? 0) * 100) * Math.max(nights, 0) * Math.max(numRooms, 1);
