import { precacheAndRoute } from 'workbox-precaching';

// Precache de los assets de la app (lo genera vite-plugin-pwa en build).
precacheAndRoute(self.__WB_MANIFEST);

// Con strategies:'injectManifest' (Service Worker propio, no autogenerado)
// este listener NO viene incluido gratis como con generateSW -- hay que
// agregarlo a mano. Sin él, ActualizarApp.jsx podía mostrar el aviso de
// "hay versión nueva" y pedir que la apliques, pero el botón "Actualizar"
// no hacía nada de verdad: el Service Worker nuevo se quedaba esperando
// (WAITING) hasta que la persona cerrara TODAS las pestañas/ventanas de la
// app, cosa que casi nunca pasa en una PWA que se queda abierta días
// (la tablet del kiosco, sobre todo). Este mensaje es justo lo que
// updateServiceWorker(true) del hook manda para activar la espera.
self.addEventListener('message', (event) => {
  if (event.data?.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});

// clients.claim(): en cuanto el Service Worker nuevo activa, toma control
// de las pestañas ya abiertas de inmediato -- sin esto, una pestaña que
// sigue abierta seguiría sirviéndose del Service Worker viejo hasta su
// próxima recarga completa, aunque ya haya "aceptado" la actualización.
self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = { titulo: 'Pasitos', cuerpo: event.data ? event.data.text() : '' };
  }

  event.waitUntil(
    self.registration.showNotification(data.titulo || 'Pasitos', {
      body: data.cuerpo || '',
      icon: '/logo192.png',
      badge: '/logo192.png',
      data: { url: data.url || '/' },
    })
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil(self.clients.openWindow(event.notification.data?.url || '/'));
});
