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
      injectRegister: 'auto', // Registra el Service Worker automáticamente
      includeAssets: ['favicon.ico', 'logo192.png', 'logo512.png'],
      manifest: {
        short_name: "Guardería",
        name: "Sistema Biométrico Guardería",
        display: "standalone",
        theme_color: "#7c3aed",
        background_color: "#ffffff",
        orientation: "portrait",
        icons: [
          {
            src: "logo192.png",
            type: "image/png",
            sizes: "192x192"
          },
          {
            src: "logo512.png",
            type: "image/png",
            sizes: "512x512",
            purpose: "any maskable" // Optimiza para Android
          }
        ]
      }
    })
  ],
  server: {
    https: true, // <-- CAMBIAR A TRUE para que funcione la cámara y PWA
    host: true,
    port: 5173
  }
})