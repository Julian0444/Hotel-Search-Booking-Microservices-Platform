/**
 * Prefetch intencional (plan 13 fase 9, FE2): al hover/focus de una card o
 * de "View hotel" se precarga SOLO el chunk lazy de HotelDetail — la
 * navegación más probable desde un listado — nunca el sitio entero.
 * import() es idempotente (el module graph cachea), pero el latch evita
 * crear promesas de más en cada hover; si la descarga falla (offline) se
 * rearma para que la navegación real la reintente.
 */

let requested = false;

export const prefetchHotelDetailChunk = () => {
  if (requested) return;
  requested = true;
  import('../pages/HotelDetail').catch(() => {
    requested = false;
  });
};
