import { describe, expect, it } from 'vitest';
import {
  RESERVATION_GROUPS,
  canCancelReservation,
  reservationCounts,
  reservationDisplayStatus,
  reservationGroup,
} from './reservations';

const TODAY = '2026-08-02';
const confirmed = (checkIn, checkOut) => ({ status: 'confirmed', check_in: checkIn, check_out: checkOut });
const cancelled = (checkIn, checkOut) => ({ status: 'cancelled', check_in: checkIn, check_out: checkOut });

describe('reservationGroup (F13-06: status del backend manda)', () => {
  it('cancelled va a Cancelled aunque sea futura', () => {
    expect(reservationGroup(cancelled('2027-01-01', '2027-01-05'), TODAY)).toBe(RESERVATION_GROUPS.CANCELLED);
  });

  it('confirmada terminada → Past; futura o en curso → Upcoming', () => {
    expect(reservationGroup(confirmed('2026-01-05', '2026-01-08'), TODAY)).toBe(RESERVATION_GROUPS.PAST);
    expect(reservationGroup(confirmed('2027-03-10', '2027-03-13'), TODAY)).toBe(RESERVATION_GROUPS.UPCOMING);
    expect(reservationGroup(confirmed('2026-08-01', '2026-08-04'), TODAY)).toBe(RESERVATION_GROUPS.UPCOMING);
  });
});

describe('reservationDisplayStatus', () => {
  it('deriva la etiqueta visual de status + fechas', () => {
    expect(reservationDisplayStatus(cancelled('2027-01-01', '2027-01-05'), TODAY).label).toBe('Cancelled');
    expect(reservationDisplayStatus(confirmed('2026-01-05', '2026-01-08'), TODAY).label).toBe('Completed');
    expect(reservationDisplayStatus(confirmed('2026-08-01', '2026-08-04'), TODAY).label).toBe('In progress');
    expect(reservationDisplayStatus(confirmed('2027-03-10', '2027-03-13'), TODAY).label).toBe('Confirmed');
  });
});

describe('canCancelReservation (lifecycle, no reloj)', () => {
  it('solo confirmed es cancelable; el backend decide el resto', () => {
    expect(canCancelReservation(confirmed('2026-01-05', '2026-01-08'))).toBe(true);
    expect(canCancelReservation(cancelled('2027-01-01', '2027-01-05'))).toBe(false);
  });
});

describe('reservationCounts', () => {
  it('cuenta por tab sin ocultar nada', () => {
    const counts = reservationCounts(
      [confirmed('2027-03-10', '2027-03-13'), confirmed('2026-01-05', '2026-01-08'), cancelled('2026-05-01', '2026-05-03')],
      TODAY,
    );
    expect(counts).toEqual({ all: 3, upcoming: 1, past: 1, cancelled: 1 });
  });
});
