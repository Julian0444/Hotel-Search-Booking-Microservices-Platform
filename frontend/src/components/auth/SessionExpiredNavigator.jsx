/**
 * Escucha session-expired DENTRO del Router (plan 13 fase 2): navega a
 * /login preservando la ubicación actual en state.from para volver tras
 * reautenticar. Reemplaza el hard redirect que hacía el interceptor viejo.
 */

import { useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { onSessionExpired } from '../../services/authEvents';
import { ROUTES } from '../../constants';
import { useAuth } from '../../hooks/useAuth';

const SessionExpiredNavigator = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { logout } = useAuth();

  // Sign out primero solicita una navegación normal. Sólo después de que
  // useBlocker permite salir se limpia la sesión; así no desmonta un form sucio.
  useEffect(() => {
    if (location.state?.signOut) {
      logout();
      navigate(ROUTES.HOME, { replace: true, state: null });
    }
  }, [location.state, logout, navigate]);

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
