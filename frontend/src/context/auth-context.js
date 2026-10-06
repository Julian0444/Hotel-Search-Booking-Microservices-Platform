/**
 * Context primitivo de auth (plan 13 fase 2): separado del provider para que
 * AuthContext.jsx exporte SOLO componentes — eso resuelve el error real de
 * react-refresh/only-export-components sin apagar la regla.
 */

import { createContext } from 'react';

export const AuthContext = createContext(null);
