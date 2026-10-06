/**
 * Base API configuration
 * Instancia de Axios con auth y normalización de errores (plan 13, F13-01/02).
 * Todo lo que sale de acá rechaza con ApiError; el 401 de sesión emite el
 * evento session-expired (nada de hard redirects del navegador — RV31).
 */

import axios from 'axios';
import { API_CONFIG } from '../constants';
import { toApiError, isAbortError } from './apiError';
import { readToken, clearSession } from './authStorage';
import { emitSessionExpired } from './authEvents';

const api = axios.create({
  baseURL: API_CONFIG.BASE_URL,
  timeout: API_CONFIG.TIMEOUT,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = readToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Cancelaciones de AbortSignal: se propagan tal cual para que TanStack
    // Query las ignore (no son errores de UI ni de sesión)
    if (isAbortError(error)) {
      return Promise.reject(error);
    }

    // Un 401 del propio login es "credenciales inválidas", no sesión
    // expirada: debe llegar al formulario (RV11). Un 401 de cualquier otra
    // ruta CON token guardado significa sesión vencida/revocada: se limpia
    // el storage y se emite el evento — AuthProvider limpia estado y el
    // Router navega a /login preservando la ubicación (F13-02).
    const isLoginRequest = error.config?.url?.includes('/login');
    if (error.response?.status === 401 && !isLoginRequest && readToken()) {
      clearSession();
      emitSessionExpired();
    }

    return Promise.reject(toApiError(error));
  },
);

export default api;
