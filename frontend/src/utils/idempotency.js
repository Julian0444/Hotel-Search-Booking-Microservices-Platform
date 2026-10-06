/**
 * Idempotencia por intento de usuario (plan 13, F13-05).
 * Un "intento" es un payload concreto: mientras el usuario reintente el
 * MISMO payload (timeout, 5xx, red), la key se conserva y hotels-api
 * dedupea (A3). Si cambia fechas/rooms/guests es un intento nuevo → key
 * nueva. Nunca una key global ni por sesión.
 */

export const createAttemptTracker = (generateKey = () => crypto.randomUUID()) => {
  let currentKey = null;
  let currentFingerprint = null;

  return {
    /** Key estable para este payload; rota sola cuando el payload cambia. */
    keyFor(payload) {
      const fingerprint = JSON.stringify(payload);
      if (fingerprint !== currentFingerprint) {
        currentFingerprint = fingerprint;
        currentKey = generateKey();
      }
      return currentKey;
    },

    /** Tras un intento CONFIRMADO (201/409 de negocio): el próximo submit es otro intento. */
    reset() {
      currentKey = null;
      currentFingerprint = null;
    },
  };
};
