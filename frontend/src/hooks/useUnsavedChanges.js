/**
 * useUnsavedChanges (plan 13 fase 7, F13-08): con cambios sucios,
 * - cerrar/recargar el tab dispara el prompt nativo (beforeunload);
 * - las salidas propias del form (Cancel/Back) pasan por confirmLeave().
 * (useBlocker de react-router exige data router; el app usa BrowserRouter
 * declarativo, así que las salidas in-app se interceptan en los botones.)
 */

import { useEffect, useState } from 'react';

export const useUnsavedChanges = (isDirty) => {
  const [confirming, setConfirming] = useState(null); // () => void pendiente

  useEffect(() => {
    if (!isDirty) return undefined;
    const handler = (event) => {
      event.preventDefault();
      // Requerido por el estándar para disparar el prompt
      event.returnValue = '';
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [isDirty]);

  /** Envuelve una salida: con cambios sucios abre el dialog; limpio, sale. */
  const confirmLeave = (leave) => {
    if (isDirty) {
      setConfirming(() => leave);
    } else {
      leave();
    }
  };

  return {
    confirmLeave,
    isConfirming: confirming !== null,
    stay: () => setConfirming(null),
    discardAndLeave: () => {
      const leave = confirming;
      setConfirming(null);
      leave?.();
    },
  };
};
