// urlBase64ToUint8Array: helper estándar para convertir la VAPID public key
// (base64 URL-safe) al formato Uint8Array que pide pushManager.subscribe().
function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = atob(base64);
  return Uint8Array.from([...rawData].map((c) => c.charCodeAt(0)));
}

function conTimeout(promesa, ms, mensaje) {
  return Promise.race([
    promesa,
    new Promise((_, reject) => setTimeout(() => reject(new Error(mensaje)), ms)),
  ]);
}

// pushSoportado indica si el navegador puede recibir Web Push (Safari en iPhone
// solo lo soporta si la página se agregó a la pantalla de inicio).
export function pushSoportado() {
  return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
}

// suscripcionActiva verifica si YA existe una suscripción push registrada en
// este navegador, sin pedir permiso ni crear una nueva. El permiso del sistema
// operativo (Notification.permission) puede estar en "granted" sin que exista
// una suscripción real (por ejemplo, si el registro nunca se completó) — por
// eso no basta con mirar Notification.permission para decidir si ya está activo.
//
// "navigator.serviceWorker.ready" nunca se resuelve si no hay Service Worker
// activo (por ejemplo, en "npm run dev" sin devOptions.enabled, o si el
// registro falló) — por eso esta función SIEMPRE debe llamarse con timeout y
// nunca debe usarse para bloquear la carga inicial de una pantalla.
export async function suscripcionActiva() {
  if (!pushSoportado()) return false;
  if (Notification.permission !== 'granted') return false;
  try {
    const registration = await conTimeout(navigator.serviceWorker.ready, 3000, 'sin Service Worker activo');
    const subscription = await registration.pushManager.getSubscription();
    return !!subscription;
  } catch {
    return false;
  }
}

// suscribirseAPush pide permiso de notificaciones y, si se concede, registra la
// suscripción en el backend. Devuelve true si quedó activa.
export async function suscribirseAPush(api) {
  if (!pushSoportado()) return false;

  const permiso = await Notification.requestPermission();
  if (permiso !== 'granted') return false;

  const vapidKey = import.meta.env.VITE_VAPID_PUBLIC_KEY;
  if (!vapidKey) {
    console.warn('VITE_VAPID_PUBLIC_KEY no configurada; no se puede suscribir a push.');
    return false;
  }

  const registration = await conTimeout(
    navigator.serviceWorker.ready,
    10000,
    'No hay un Service Worker activo todavía (recarga la página e inténtalo de nuevo).'
  );
  let subscription = await registration.pushManager.getSubscription();
  if (!subscription) {
    subscription = await conTimeout(
      registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidKey),
      }),
      15000,
      'El navegador tardó demasiado en registrar la suscripción (revisa tu conexión).'
    );
  }

  await api.post('/push/suscribir', subscription.toJSON());
  return true;
}
