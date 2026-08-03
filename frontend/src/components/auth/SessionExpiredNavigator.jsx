/**
 * Escucha session-expired DENTRO del Router (plan 13 fase 2): navega a
 * /login preservando la ubicación actual en state.from para volver tras
 * reautenticar. Reemplaza el hard redirect que hacía el interceptor viejo.
 */

import { useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { onSessionExpired } from '../../services/authEvents';
import { ROUTES } from '../../constants';

const SessionExpiredNavigator = () => {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(
    () =>
      onSessionExpired(() => {
        if (location.pathname !== ROUTES.LOGIN) {
          navigate(ROUTES.LOGIN, { replace: true, state: { from: location } });
        }
      }),
    [navigate, location],
  );

  return null;
};

export default SessionExpiredNavigator;
