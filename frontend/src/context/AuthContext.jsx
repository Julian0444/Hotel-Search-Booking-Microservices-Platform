/**
 * Authentication Context
 * Provides authentication state and methods throughout the application
 */

import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { authService } from '../services';
import { STORAGE_KEYS, USER_ROLES } from '../constants';

const AuthContext = createContext(null);

/**
 * Custom hook to access auth context
 * @returns {AuthContextValue}
 */
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

/**
 * Auth Provider Component
 */
export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Load user from localStorage on mount
  useEffect(() => {
    const storedUser = localStorage.getItem(STORAGE_KEYS.USER);
    const storedToken = localStorage.getItem(STORAGE_KEYS.TOKEN);

    if (storedUser && storedToken) {
      try {
        setUser(JSON.parse(storedUser));
      } catch {
        localStorage.removeItem(STORAGE_KEYS.USER);
        localStorage.removeItem(STORAGE_KEYS.TOKEN);
      }
    }
    setLoading(false);
  }, []);

  /**
   * Login user
   * @param {string} username
   * @param {string} password
   * @returns {Promise<{ success: boolean, error?: string }>}
   */
  const login = useCallback(async (username, password) => {
    setError(null);
    setLoading(true);

    try {
      const response = await authService.login(username, password);
      const userData = {
        id: response.user_id,
        username: response.username,
        tipo: response.tipo,
      };

      localStorage.setItem(STORAGE_KEYS.TOKEN, response.token);
      localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(userData));
      setUser(userData);

      return { success: true };
    } catch (err) {
      const message = err.response?.data?.error || 'Failed to login. Please try again.';
      setError(message);
      return { success: false, error: message };
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * Register new user
   * @param {string} username
   * @param {string} password
   * @param {string} tipo
   * @returns {Promise<{ success: boolean, error?: string }>}
   */
  const register = useCallback(async (username, password, tipo = USER_ROLES.CLIENT) => {
    setError(null);
    setLoading(true);

    try {
      await authService.register(username, password, tipo);
      // Auto-login after registration
      return await login(username, password);
    } catch (err) {
      const message = err.response?.data?.error || 'Registration failed. Please try again.';
      setError(message);
      return { success: false, error: message };
    } finally {
      setLoading(false);
    }
  }, [login]);

  /**
   * Logout user
   */
  const logout = useCallback(() => {
    localStorage.removeItem(STORAGE_KEYS.TOKEN);
    localStorage.removeItem(STORAGE_KEYS.USER);
    setUser(null);
    setError(null);
  }, []);

  const isAdmin = user?.tipo === USER_ROLES.ADMIN;
  const isAuthenticated = !!user;

  const value = {
    user,
    loading,
    error,
    isAuthenticated,
    isAdmin,
    login,
    register,
    logout,
    setError,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export default AuthContext;
