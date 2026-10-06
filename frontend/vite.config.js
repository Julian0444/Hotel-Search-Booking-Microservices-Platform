import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      // El gateway expone la API versionada bajo /api/v1: se proxya tal cual,
      // sin recortar el prefijo (plan 07 / A2).
      // TLS local (plan 10): el gateway escucha en 443 con cert self-signed —
      // secure:false para que el proxy de Vite no rechace ese cert; el browser
      // sigue hablando http plano con el dev server.
      '/api': {
        target: 'https://localhost',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  preview: {
    port: 5173,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        // FE2 (medido en fase 9): MUI ya NO va a un mega-vendor precargado —
        // Rollup reparte cada módulo según uso real y los componentes que solo
        // usan rutas lazy (Tabs, Table, Autocomplete, ImageList…) salen del
        // payload inicial (el vendor-mui completo sumaba ~850ms de JS sin usar
        // en Lighthouse mobile). En chunks nombrados va SOLO lo que el entry
        // importa estático de todas formas (se precarga igual: neutro en
        // payload) para que ningún chunk pase los 500 kB: react/router,
        // react-query/axios y el runtime de emotion.
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined;
          if (/[\\/]node_modules[\\/](react|react-dom|scheduler|react-router)[\\/]/.test(id)) {
            return 'vendor-react';
          }
          if (/[\\/]node_modules[\\/](@tanstack|axios)[\\/]/.test(id)) {
            return 'vendor-data';
          }
          if (/[\\/]node_modules[\\/]@emotion[\\/]/.test(id)) {
            return 'vendor-emotion';
          }
          return undefined;
        },
      },
    },
  },
  // Unit/component tests (plan 13 fase 0): jsdom + MSW; los e2e de Playwright
  // viven en e2e/ y corren aparte (npm run test:e2e)
  test: {
    environment: 'jsdom',
    globals: false,
    setupFiles: ['./src/test/setup.js'],
    include: ['src/**/*.test.{js,jsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/**/*.{js,jsx}'],
      exclude: [
        'src/test/**',
        'src/**/*.test.{js,jsx}',
        'src/main.jsx',
        'src/types/**',
      ],
      thresholds: {
        statements: 60,
        branches: 50,
        functions: 60,
        lines: 60,
        // Los módulos de riesgo (contrato, fechas, sesión) exigen más
        'src/services/**': { statements: 80, lines: 80 },
        'src/utils/**': { statements: 80, lines: 80 },
        'src/context/**': { statements: 80, lines: 80 },
      },
    },
  },
});
