/**
 * Presentación del historial de reservas (plan 13, F13-06).
 * El backend es el dueño del lifecycle: `status` manda (confirmed|cancelled);
 * las fechas civiles solo ubican una confirmada en el tiempo para agrupar.
 */

import { RESERVATION_STATUS } from '../constants';
import { compareDateOnly, todayLocal } from './dateOnly';

export const RESERVATION_GROUPS = {
  UPCOMING: 'upcoming',
  PAST: 'past',
  CANCELLED: 'cancelled',
};

/** Grupo temporal de una reserva (para tabs); nunca oculta nada. */
export const reservationGroup = (reservation, today = todayLocal()) => {
  if (reservation.status === RESERVATION_STATUS.CANCELLED) return RESERVATION_GROUPS.CANCELLED;
  if (compareDateOnly(reservation.check_out, today) < 0) return RESERVATION_GROUPS.PAST;
  return RESERVATION_GROUPS.UPCOMING;
};

/**
 * Chip de estado: deriva la etiqueta VISUAL de status + fechas.
 * cancelled → Cancelled; confirmada terminada → Completed; en curso →
 * In progress; futura → Confirmed.
 */
export const reservationDisplayStatus = (reservation, today = todayLocal()) => {
  if (reservation.status === RESERVATION_STATUS.CANCELLED) {
    return { label: 'Cancelled', color: 'default' };
  }
  if (compareDateOnly(reservation.check_out, today) < 0) {
    return { label: 'Completed', color: 'info' };
  }
  if (compareDateOnly(reservation.check_in, today) <= 0) {
    return { label: 'In progress', color: 'success' };
  }
  return { label: 'Confirmed', color: 'primary' };
};

/**
 * Cancelable = el lifecycle lo permite (status confirmed). El reloj del
 * browser NO decide permisos: hotels-api valida ownership y estado.
 */
export const canCancelReservation = (reservation) =>
  reservation.status === RESERVATION_STATUS.CONFIRMED;

/** Counts por tab a partir de la lista completa. */
export const reservationCounts = (reservations, today = todayLocal()) => {
  const counts = { all: reservations.length, upcoming: 0, past: 0, cancelled: 0 };
  for (const reservation of reservations) {
    counts[reservationGroup(reservation, today)] += 1;
  }
  return counts;
};
