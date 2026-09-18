import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: 3000,
    proxy: {
      // Send API calls to the Go service so the browser sees one origin.
      //
      // This is not just to avoid configuring CORS. The session lives in a
      // SameSite=Lax cookie: talking to http://localhost:8080 directly from a
      // page on :3000 is a cross-site request, and the browser would withhold
      // the cookie from every mutation. Proxying keeps everything same-origin,
      // which is also how this deploys -- the SPA and the API behind one host.
      '/v1': {
        target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: false
      }
    }
  }
});
