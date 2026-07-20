import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import basicSsl from '@vitejs/plugin-basic-ssl'
import { VitePWA } from 'vite-plugin-pwa' // <-- IMPORTANTE

export default defineConfig({
  plugins: [
    react(),
    basicSsl(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'auto', // Registra el Service Worker automáticamente (única fuente de registro)
      includeAssets: ['favicon.ico', 'logo192.png', 'logo512.png'],
      // El manifest se sirve de forma estática desde public/manifest.json y se
      // enlaza explícitamente en index.html. Desactivamos la generación/inyección
      // propia del plugin para no terminar con dos <link rel="manifest"> distintos.
      manifest: false
    })
  ],
  server: {
    https: true, // <-- CAMBIAR A TRUE para que funcione la cámara y PWA
    host: true,
    port: 5173
  }
})