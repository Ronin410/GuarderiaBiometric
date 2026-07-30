import { precacheAndRoute } from 'workbox-precaching';

// Precache de los assets de la app (lo genera vite-plugin-pwa en build).
precacheAndRoute(self.__WB_MANIFEST);

self.addEventListener('push', (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = { titulo: 'BioSafe', cuerpo: event.data ? event.data.text() : '' };
  }

  event.waitUntil(
    self.registration.showNotification(data.titulo || 'BioSafe', {
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
