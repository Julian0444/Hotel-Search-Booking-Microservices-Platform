/**
 * Eventos de sesión (plan 13 fase 2, RV31): el interceptor HTTP no navega ni
 * recarga — emite `session-expired` y quien tenga Router/estado decide.
 * Antes: hard redirect del navegador a /login (recarga que perdía state.from
 * y rompía la continuidad SPA).
 */

const target = new EventTarget();

export const SESSION_EXPIRED_EVENT = 'session-expired';

export const emitSessionExpired = () => {
  target.dispatchEvent(new Event(SESSION_EXPIRED_EVENT));
};

/** @returns {() => void} unsubscribe */
export const onSessionExpired = (handler) => {
  target.addEventListener(SESSION_EXPIRED_EVENT, handler);
  return () => target.removeEventListener(SESSION_EXPIRED_EVENT, handler);
};
