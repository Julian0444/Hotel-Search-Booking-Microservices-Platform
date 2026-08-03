/**
 * F13-04: las fechas civiles no dependen del huso. El CI corre esta suite con
 * TZ=America/Los_Angeles además de UTC; los casos con `now` inyectado cubren
 * el cambio de día y DST explícitamente.
 */

import { describe, expect, it } from 'vitest';
import {
  addDaysDateOnly,
  compareDateOnly,
  differenceInNights,
  formatDateOnlyLabel,
  formatTimeLabel,
  isValidDateOnly,
  todayLocal,
} from './dateOnly';

describe('todayLocal', () => {
  it('usa los componentes LOCALES del reloj, no UTC', () => {
    // 23:30 local: en LA eso es 06:30Z del día siguiente — toISOString
    // devolvería el día equivocado; todayLocal debe devolver el día local
    const lateNight = new Date(2026, 7, 2, 23, 30, 0);
    expect(todayLocal(lateNight)).toBe('2026-08-02');

    const earlyMorning = new Date(2026, 7, 3, 0, 10, 0);
    expect(todayLocal(earlyMorning)).toBe('2026-08-03');
  });
});

describe('addDaysDateOnly', () => {
  it('suma días calendario', () => {
    expect(addDaysDateOnly('2026-08-02', 1)).toBe('2026-08-03');
    expect(addDaysDateOnly('2026-08-31', 1)).toBe('2026-09-01');
    expect(addDaysDateOnly('2026-12-31', 1)).toBe('2027-01-01');
    expect(addDaysDateOnly('2026-08-02', -2)).toBe('2026-07-31');
  });

  it('cruza el cambio de DST de EEUU sin perder ni duplicar un día', () => {
    // 2026-03-08: entra el DST en America/Los_Angeles (día de 23hs locales)
    expect(addDaysDateOnly('2026-03-07', 1)).toBe('2026-03-08');
    expect(addDaysDateOnly('2026-03-08', 1)).toBe('2026-03-09');
    // 2026-11-01: sale el DST (día de 25hs locales)
    expect(addDaysDateOnly('2026-10-31', 2)).toBe('2026-11-02');
  });
});

describe('differenceInNights', () => {
  it('cuenta noches exactas', () => {
    expect(differenceInNights('2027-03-10', '2027-03-13')).toBe(3);
    expect(differenceInNights('2026-08-02', '2026-08-03')).toBe(1);
  });

  it('no cambia por DST (una estadía que cruza el cambio de hora)', () => {
    expect(differenceInNights('2026-03-07', '2026-03-09')).toBe(2);
    expect(differenceInNights('2026-10-31', '2026-11-02')).toBe(2);
  });
});

describe('isValidDateOnly / compareDateOnly', () => {
  it('acepta fechas calendario reales y rechaza el resto', () => {
    expect(isValidDateOnly('2026-08-02')).toBe(true);
    expect(isValidDateOnly('2026-02-30')).toBe(false);
    expect(isValidDateOnly('2026-13-01')).toBe(false);
    expect(isValidDateOnly('02-08-2026')).toBe(false);
    expect(isValidDateOnly('')).toBe(false);
    expect(isValidDateOnly(undefined)).toBe(false);
  });

  it('compara cronológicamente', () => {
    expect(compareDateOnly('2026-08-01', '2026-08-02')).toBe(-1);
    expect(compareDateOnly('2026-08-02', '2026-08-02')).toBe(0);
    expect(compareDateOnly('2026-09-01', '2026-08-02')).toBe(1);
  });
});

describe('formatDateOnlyLabel', () => {
  it('muestra EXACTAMENTE el día del string en cualquier huso', () => {
    // En LA, new Date('2027-03-10') sería Mar 9 local — la etiqueta debe
    // decir Mar 10 igual
    expect(formatDateOnlyLabel('2027-03-10')).toBe('Mar 10, 2027');
  });

  it('devuelve el input crudo si no es una fecha civil', () => {
    expect(formatDateOnlyLabel('n/a')).toBe('n/a');
    expect(formatDateOnlyLabel('')).toBe('');
  });
});

describe('formatTimeLabel', () => {
  it('convierte el "HH:mm" del contrato en etiqueta legible, nunca RFC3339', () => {
    expect(formatTimeLabel('15:00')).toBe('3:00 PM');
    expect(formatTimeLabel('10:00')).toBe('10:00 AM');
    expect(formatTimeLabel('00:30')).toBe('12:30 AM');
  });

  it('no inventa nada con inputs fuera de contrato', () => {
    expect(formatTimeLabel('')).toBe('');
    expect(formatTimeLabel('2024-01-01T15:00:00Z')).toBe('2024-01-01T15:00:00Z');
  });
});
