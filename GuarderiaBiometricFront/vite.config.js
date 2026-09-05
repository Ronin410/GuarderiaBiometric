import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import basicSsl from '@vitejs/plugin-basic-ssl'
import { VitePWA } from 'vite-plugin-pwa' // <-- IMPORTANTE

export default defineConfig({
  plugins: [
    react(),
    basicSsl(),
    VitePWA({
      // 'prompt' (en vez de 'autoUpdate'): cuando hay versión nueva, NO se
      // recarga sola de golpe (le borraría a un papá a medio llenar un
      // formulario, o a una tablet de recepción a medio escaneo) -- en vez
      // de eso, ActualizarApp.jsx (montado en main.jsx, visible en toda la
      // app) muestra un aviso con botón "Actualizar" y aplica la nueva
      // versión solo cuando alguien lo confirma.
      registerType: 'prompt',
      // El registro del Service Worker lo hace el hook useRegisterSW() de
      // ActualizarApp.jsx (virtual:pwa-register/react) -- injectRegister
      // en false evita registrarlo una segunda vez desde un <script>
      // inyectado aparte.
      injectRegister: false,
      includeAssets: ['favicon.png', 'logo192.png', 'logo512.png', 'apple-touch-icon.png'],
      // El manifest se sirve de forma estática desde public/manifest.json y se
      // enlaza explícitamente en index.html. Desactivamos la generación/inyección
      // propia del plugin para no terminar con dos <link rel="manifest"> distintos.
      manifest: false,
      // Service Worker propio (src/sw.js) en vez de uno autogenerado, para poder
      // escuchar los eventos "push" y "notificationclick" de las notificaciones.
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.js'
    })
  ],
  server: {
    https: true, // <-- CAMBIAR A TRUE para que funcione la cámara y PWA
    host: true,
    port: 5173
  }
})