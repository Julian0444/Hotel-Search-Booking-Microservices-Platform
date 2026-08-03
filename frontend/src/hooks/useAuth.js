/**
 * useAuth (plan 13 fase 2): consume el context primitivo. Vive fuera de
 * AuthContext.jsx para que ese archivo exporte solo componentes (Fast
 * Refresh limpio).
 */

import { useContext } from 'react';
import { AuthContext } from '../context/auth-context';

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
