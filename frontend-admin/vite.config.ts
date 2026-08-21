import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Overridable at build time (see Dockerfile) so this same build can be served
  // either at the domain root (local/default) or under a path prefix like
  // /admin/ when path-based routing shares one host with the main frontend.
  base: process.env.VITE_BASE_PATH || '/',
  server: {
    port: Number(process.env.PORT) || 5174,
    proxy: {
      // admin is the admin-only, unauthenticated backoffice service (see backend/cmd/admin).
      '/api': { target: 'http://localhost:3001', changeOrigin: true, rewrite: (path) => path.replace(/^\/api/, '') },
    },
  },
})
