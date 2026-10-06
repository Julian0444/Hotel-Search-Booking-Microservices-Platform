/**
 * Auth Provider (plan 13 fase 2, RV31/F13-02).
 * - Bootstrap: valida exp/iss/aud del token guardado ANTES de aceptar la
 *   sesión (adiós sesión zombie); malformado/vencido arranca anónimo.
 * - isBootstrapping (carga inicial) e isSubmitting (login/register en vuelo)
 *   son estados separados: el loader de app no es el spinner del form.
 * - El interceptor emite session-expired; acá solo se limpia estado — el
 *   componente de rutas navega con el Router (sin hard redirect del navegador).
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { AuthContext } from './auth-context';
import { authService } from '../services';
import { persistSession, clearSession, readStoredSession } from '../services/authStorage';
import { onSessionExpired } from '../services/authEvents';
import { USER_ROLES } from '../constants';

export const AuthProvider = ({ children }) => {
  const queryClient = useQueryClient();
  const [user, setUser] = useState(null);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [sessionNotice, setSessionNotice] = useState(null);

  useEffect(() => {
    const session = readStoredSession();
    if (session) {
      setUser(session.user);
    }
    setIsBootstrapping(false);
  }, []);

  // Un 401 de sesión en cualquier request: el storage ya quedó limpio
  // (interceptor); acá cae el estado y se anota el motivo para que Login lo
  // explique. Las queries privadas se descartan para que Back no muestre
  // datos de la sesión anterior.
  useEffect(
    () =>
      onSessionExpired(() => {
        setUser(null);
        setSessionNotice('Your session expired. Please sign in again.');
        queryClient.clear();
      }),
    [queryClient],
  );

  const login = useCallback(async (username, password) => {
    setIsSubmitting(true);
    try {
      const response = await authService.login(username, password);
      const userData = {
        id: String(response.user_id),
        username: response.username,
        tipo: response.tipo,
      };
      persistSession({ token: response.token, user: userData });
      setUser(userData);
      setSessionNotice(null);
      return { success: true };
    } catch (error) {
      return { success: false, error: error.message };
    } finally {
      setIsSubmitting(false);
    }
  }, []);

  const register = useCallback(
    async (username, password) => {
      setIsSubmitting(true);
      try {
        await authService.register(username, password);
      } catch (error) {
        setIsSubmitting(false);
        return { success: false, error: error.message };
      }
      // Auto-login after registration
      return login(username, password);
    },
    [login],
  );

  const logout = useCallback(() => {
    clearSession();
    setUser(null);
    setSessionNotice(null);
    // Invalida TODO el estado remoto privado: reservas, listados admin, etc.
    queryClient.clear();
  }, [queryClient]);

  const value = useMemo(
    () => ({
      user,
      isBootstrapping,
      isSubmitting,
      sessionNotice,
      clearSessionNotice: () => setSessionNotice(null),
      isAuthenticated: !!user,
      isAdmin: user?.tipo === USER_ROLES.ADMIN,
      login,
      register,
      logout,
    }),
    [user, isBootstrapping, isSubmitting, sessionNotice, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
