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
      // sin recortar el prefijo (plan 07 / A2)
      '/api': {
        target: 'http://localhost',
        changeOrigin: true,
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
