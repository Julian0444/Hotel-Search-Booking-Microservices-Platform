/**
 * Servidor MSW para Node/vitest (plan 13 fase 0).
 */

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
