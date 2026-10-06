import { describe, expect, it } from 'vitest';
import { isHttpUrl, isOptionalEmail, isTimeOfDay } from './validators';
import { amenityLabel, getInitials, hotelImage, shortId, truncateText } from './helpers';
import { HOTEL_FALLBACK_IMAGE } from '../constants';

describe('isHttpUrl (F13-08: URLs de imagen)', () => {
  it('acepta solo http(s) absolutas', () => {
    expect(isHttpUrl('https://images.example/a.jpg')).toBe(true);
    expect(isHttpUrl('http://images.example/a.jpg')).toBe(true);
    expect(isHttpUrl('javascript:alert(1)')).toBe(false);
    expect(isHttpUrl('data:image/png;base64,AAAA')).toBe(false);
    expect(isHttpUrl('/relative/path.jpg')).toBe(false);
    expect(isHttpUrl('')).toBe(false);
  });
});

describe('isTimeOfDay (contrato "HH:mm", RV21)', () => {
  it('espejo del validTimeOfDay de hotels-api', () => {
    expect(isTimeOfDay('15:00')).toBe(true);
    expect(isTimeOfDay('00:00')).toBe(true);
    expect(isTimeOfDay('23:59')).toBe(true);
    expect(isTimeOfDay('24:00')).toBe(false);
    expect(isTimeOfDay('9:00')).toBe(false);
    expect(isTimeOfDay('2024-01-01T15:00:00Z')).toBe(false);
    expect(isTimeOfDay('')).toBe(false);
  });
});

describe('isOptionalEmail', () => {
  it('vacío es válido (campo opcional del contrato)', () => {
    expect(isOptionalEmail('')).toBe(true);
    expect(isOptionalEmail('reservas@hotel.example')).toBe(true);
    expect(isOptionalEmail('not-an-email')).toBe(false);
  });
});

describe('helpers de presentación', () => {
  it('hotelImage usa la primera imagen o el fallback local', () => {
    expect(hotelImage({ images: ['https://x/a.jpg'] })).toBe('https://x/a.jpg');
    expect(hotelImage({ images: [] })).toBe(HOTEL_FALLBACK_IMAGE);
    expect(hotelImage(undefined)).toBe(HOTEL_FALLBACK_IMAGE);
  });

  it('amenityLabel normaliza snake_case y libres', () => {
    expect(amenityLabel('air_conditioning')).toBe('Air conditioning');
    expect(amenityLabel('wifi')).toBe('Wifi');
    expect(amenityLabel('')).toBe('');
  });

  it('shortId y getInitials y truncateText', () => {
    expect(shortId('a1b2c3d4e5f60718293a4b5c')).toBe('293A4B5C');
    expect(getInitials('julian')).toBe('J');
    expect(getInitials('')).toBe('?');
    expect(truncateText('abc', 100)).toBe('abc');
    expect(truncateText('x'.repeat(120), 10)).toHaveLength(11);
  });
});
