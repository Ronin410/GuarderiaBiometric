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
// suscripción en el backend. Devuelve { ok, mensaje }: mensaje viene listo
// para mostrar cuando ok es false -- antes se regresaba solo un booleano y
// cada pantalla mostraba el MISMO mensaje genérico ("revisa los permisos de
// tu navegador") sin importar la causa real, que es justo el caso más común
// de "antes sí podía activarlas, ahora no me deja": el navegador recuerda un
// rechazo anterior y, pasado cierto número de veces, deja de mostrar el
// cuadro de permiso por completo -- Notification.requestPermission()
// entonces resuelve 'denied' en silencio, sin volver a preguntar nada, y no
// hay forma de "reactivarlo" desde JavaScript. Detectarlo ANTES de llamar a
// requestPermission() es lo único que permite distinguirlo de un rechazo
// nuevo y decirle a la persona exactamente qué hacer (entrar a los ajustes
// del sitio en el navegador), en vez de un mensaje que suena a que la app
// está rota.
//
// ruta es opcional -- por defecto es la de un papá/staff con sesión normal
// (/push/suscribir); /plataforma la manda como '/plataforma/push/suscribir'
// porque esa sesión se autentica con X-Platform-Key, no con el JWT normal
// (ver PanelPlataforma.jsx).
export async function suscribirseAPush(api, ruta = '/push/suscribir') {
  if (!pushSoportado()) {
    return { ok: false, mensaje: 'Este navegador no soporta notificaciones push.' };
  }

  if (Notification.permission === 'denied') {
    return {
      ok: false,
      mensaje: 'Ya habías bloqueado las notificaciones para esta app en tu navegador, así que no se puede volver a pedir el permiso desde aquí. Entra a los ajustes del sitio de tu navegador (el candado o los tres puntos junto a la dirección), busca "Notificaciones" y cámbialo a "Permitir".',
    };
  }

  const permiso = await Notification.requestPermission();
  if (permiso !== 'granted') {
    return { ok: false, mensaje: 'No se concedió el permiso de notificaciones.' };
  }

  const vapidKey = import.meta.env.VITE_VAPID_PUBLIC_KEY;
  if (!vapidKey) {
    console.warn('VITE_VAPID_PUBLIC_KEY no configurada; no se puede suscribir a push.');
    return { ok: false, mensaje: 'Las notificaciones no están configuradas en el servidor todavía.' };
  }

  try {
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

    await api.post(ruta, subscription.toJSON());
    return { ok: true, mensaje: '' };
  } catch (err) {
    return { ok: false, mensaje: err.message || 'No se pudieron activar las notificaciones. Inténtalo de nuevo.' };
  }
}

// desuscribirseDePush cancela la suscripción push de este navegador --
// "quiero que las notificaciones las pueda cancelar el papá con el mismo
// botón". La borra en dos lados: del navegador (pushManager.unsubscribe,
// para que dejen de llegar de verdad) y del backend (DELETE ruta, para que
// ya no intente mandarle nada a un endpoint que ya se canceló). Devuelve
// true tanto si se canceló como si ya no había ninguna suscripción activa
// (nada que hacer). ruta -- ver el comentario de suscribirseAPush.
export async function desuscribirseDePush(api, ruta = '/push/suscribir') {
  if (!pushSoportado()) return true;
  try {
    const registration = await conTimeout(navigator.serviceWorker.ready, 5000, 'sin Service Worker activo');
    const subscription = await registration.pushManager.getSubscription();
    if (!subscription) return true;

    const endpoint = subscription.endpoint;
    await subscription.unsubscribe();
    await api.delete(ruta, { data: { endpoint } });
    return true;
  } catch (err) {
    console.error('Error al desuscribirse de push:', err);
    return false;
  }
}
