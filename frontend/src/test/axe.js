/**
 * Helper de axe-core para jsdom (plan 13 fase 8): corre axe sobre el DOM
 * renderizado y falla ante violaciones serious/critical — el bar del plan.
 * color-contrast queda deshabilitado: jsdom no tiene motor de layout, así
 * que axe no puede computar contraste real acá; eso lo mide Lighthouse
 * sobre el build de producción (fase 9) y el spec de Playwright (fase 10).
 */

import { expect } from 'vitest';
import axe from 'axe-core';

const AXE_OPTIONS = {
  rules: {
    'color-contrast': { enabled: false },
  },
};

const formatViolations = (violations) =>
  violations
    .map(
      (violation) =>
        `[${violation.impact}] ${violation.id} — ${violation.help}\n` +
        violation.nodes.map((node) => `    ${node.html}`).join('\n'),
    )
    .join('\n\n');

/**
 * Corre axe sobre `context` (default: el body entero, para incluir portals
 * de MUI) y falla si hay violaciones serious o critical.
 */
export async function expectNoSeriousViolations(context = document.body) {
  const results = await axe.run(context, AXE_OPTIONS);
  const serious = results.violations.filter(
    (violation) => violation.impact === 'serious' || violation.impact === 'critical',
  );
  expect(serious, formatViolations(serious)).toEqual([]);
}
