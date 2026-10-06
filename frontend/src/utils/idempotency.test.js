import { describe, expect, it } from 'vitest';
import { createAttemptTracker } from './idempotency';

describe('createAttemptTracker (F13-05)', () => {
  const payloadA = { hotel_id: 'h1', check_in: '2027-03-10', check_out: '2027-03-13', num_rooms: 1 };
  const payloadB = { ...payloadA, num_rooms: 2 };

  it('un retry del MISMO intento conserva la key', () => {
    let n = 0;
    const tracker = createAttemptTracker(() => `key-${++n}`);
    expect(tracker.keyFor(payloadA)).toBe('key-1');
    expect(tracker.keyFor(payloadA)).toBe('key-1');
  });

  it('cambiar el payload es un intento nuevo → key nueva', () => {
    let n = 0;
    const tracker = createAttemptTracker(() => `key-${++n}`);
    expect(tracker.keyFor(payloadA)).toBe('key-1');
    expect(tracker.keyFor(payloadB)).toBe('key-2');
    expect(tracker.keyFor(payloadB)).toBe('key-2');
  });

  it('reset() fuerza key nueva aunque el payload se repita (intento confirmado)', () => {
    let n = 0;
    const tracker = createAttemptTracker(() => `key-${++n}`);
    tracker.keyFor(payloadA);
    tracker.reset();
    expect(tracker.keyFor(payloadA)).toBe('key-2');
  });

  it('por defecto genera UUIDs reales', () => {
    const tracker = createAttemptTracker();
    expect(tracker.keyFor(payloadA)).toMatch(/^[0-9a-f-]{36}$/);
  });
});
