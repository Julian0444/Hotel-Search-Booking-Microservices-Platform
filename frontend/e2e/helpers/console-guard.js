/**
 * Guardia de consola (plan 13 fase 10): los recorridos fallan ante errores de
 * consola o pageerrors no permitidos. El allowlist es explícito por test
 * (p. ej. un 429 esperable del login_limit del gateway).
 */

export const attachConsoleGuard = (page, { allow = [] } = {}) => {
  const allowPatterns = [/favicon/i, ...allow];
  const errors = [];
  page.on('console', (message) => {
    if (message.type() === 'error' && !allowPatterns.some((re) => re.test(message.text()))) {
      errors.push(message.text());
    }
  });
  page.on('pageerror', (error) => {
    errors.push(`pageerror: ${error.message}`);
  });
  return errors;
};
