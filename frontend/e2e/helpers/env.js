/**
 * Entorno de los E2E (plan 13 fase 10).
 * - El SPA buildeado corre en http://localhost:5173 y pega al gateway TLS
 *   self-signed en https://localhost (por eso ignoreHTTPSErrors en la config).
 * - Las credenciales admin son env-driven (seed al arranque de users-api):
 *   en CI vienen por env vars; local se leen del .env de la raíz del repo
 *   (ADMIN_PASSWORD local difiere del .env.example — no hardcodear).
 */

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const REPO_ROOT = fileURLToPath(new URL('../../..', import.meta.url));

export const GATEWAY_URL = process.env.E2E_GATEWAY_URL || 'https://localhost';
export const API_BASE = `${GATEWAY_URL}/api/v1`;

/** Parser mínimo de .env (KEY=VALUE, sin quoting elaborado) — evita sumar dotenv. */
const readRootDotEnv = () => {
  try {
    const raw = fs.readFileSync(path.join(REPO_ROOT, '.env'), 'utf8');
    const vars = {};
    for (const line of raw.split('\n')) {
      const match = line.match(/^\s*([A-Z0-9_]+)\s*=\s*(.*?)\s*$/);
      if (match && !line.trim().startsWith('#')) vars[match[1]] = match[2];
    }
    return vars;
  } catch {
    return {};
  }
};

export const adminCredentials = () => {
  const dotenv = readRootDotEnv();
  return {
    username: process.env.ADMIN_USERNAME || dotenv.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || dotenv.ADMIN_PASSWORD || 'change-me-min-8-chars',
  };
};

/** Cliente seedeado por la migración 0002 de users-api (idempotente). */
export const DEMO_CUSTOMER = { username: 'demo', password: 'DemoCliente123' };
