import { describe, expect, it } from 'vitest';
import { formatMajorAmount, formatMinorAmount, previewTotalCents } from './money';

describe('formatMajorAmount (price_per_night, unidades mayores)', () => {
  it('formatea precios por noche', () => {
    expect(formatMajorAmount(150.5)).toBe('$150.50');
    expect(formatMajorAmount(320)).toBe('$320');
    expect(formatMajorAmount(0)).toBe('$0');
    expect(formatMajorAmount(undefined)).toBe('$0');
  });
});

describe('formatMinorAmount (total_price, centavos)', () => {
  it('formatea centavos como moneda', () => {
    expect(formatMinorAmount(45150)).toBe('$451.50');
    expect(formatMinorAmount(30100)).toBe('$301.00');
    expect(formatMinorAmount(0)).toBe('$0.00');
    expect(formatMinorAmount(undefined)).toBe('$0.00');
  });
});

describe('previewTotalCents (espejo del cálculo de hotels-api)', () => {
  it('round(price*100) * nights * rooms', () => {
    // 150.50 → 15050 centavos × 3 noches × 1 room
    expect(previewTotalCents(150.5, 3, 1)).toBe(45150);
    expect(previewTotalCents(150.5, 2, 2)).toBe(60200);
  });

  it('redondea el precio a centavos ANTES de multiplicar (sin drift de floats)', () => {
    // 99.995 → 10000 centavos redondeados: nunca multiplicar el float por
    // noches y redondear al final (divergiría del backend)
    expect(previewTotalCents(99.995, 3, 1)).toBe(30000);
  });

  it('acota inputs raros sin NaN', () => {
    expect(previewTotalCents(100, -1, 1)).toBe(0);
    expect(previewTotalCents(100, 2, 0)).toBe(20000);
    expect(previewTotalCents(undefined, 2, 1)).toBe(0);
  });
});
