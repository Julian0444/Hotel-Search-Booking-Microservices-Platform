import { describe, expect, it } from 'vitest';
import { unwrapList, unwrapObject } from './envelope';

describe('unwrapObject', () => {
  it('devuelve body.data', () => {
    expect(unwrapObject({ data: { id: '1' } })).toEqual({ id: '1' });
    expect(unwrapObject({ data: null })).toBeNull();
  });

  it('rechaza shapes sin envelope', () => {
    expect(() => unwrapObject({ id: '1' })).toThrow(/envelope/);
    expect(() => unwrapObject(undefined)).toThrow(/envelope/);
    expect(() => unwrapObject('<html>bad gateway</html>')).toThrow(/envelope/);
  });
});

describe('unwrapList', () => {
  it('devuelve {items,total,limit,offset} desde data+meta', () => {
    const body = { data: [{ id: 'a' }], meta: { total: 45, limit: 12, offset: 24 } };
    expect(unwrapList(body)).toEqual({ items: [{ id: 'a' }], total: 45, limit: 12, offset: 24 });
  });

  it('degrada total al largo de la página cuando meta no lo trae (reservas)', () => {
    const body = { data: [{ id: 'a' }, { id: 'b' }], meta: { limit: 20, offset: 0 } };
    expect(unwrapList(body).total).toBe(2);
  });

  it('rechaza un envelope inválido', () => {
    expect(() => unwrapList({ data: { not: 'a list' } })).toThrow(/envelope/);
    expect(() => unwrapList(null)).toThrow(/envelope/);
  });
});
