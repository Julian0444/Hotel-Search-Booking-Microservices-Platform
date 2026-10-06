/** Protege salidas internas (incluido Back) con el router y recarga/cierre nativos. */
import { useEffect } from 'react';
import { useBlocker } from 'react-router';

export const useUnsavedChanges = (isDirty) => {
  const blocker = useBlocker(isDirty);
  useEffect(() => {
    if (!isDirty) return undefined;
    const handler = (event) => {
      event.preventDefault();
      event.returnValue = '';
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [isDirty]);

  return {
    confirmLeave: (leave) => leave(),
    isConfirming: blocker.state === 'blocked',
    stay: () => blocker.reset?.(),
    discardAndLeave: () => blocker.proceed?.(),
  };
};
