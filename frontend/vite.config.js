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
  },
});
