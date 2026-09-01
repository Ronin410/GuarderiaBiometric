import { Platform } from 'react-native';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import Constants from 'expo-constants';
import api from '../api/client';

// Notificaciones push nativas -- equivalente RN de utils/push.js en la web
// (que usa Web Push/VAPID para la PWA). Esta app usa el servicio de push de
// Expo en su lugar: un solo token de cadena en vez de endpoint+llaves, sin
// necesitar configuración VAPID del lado del servidor (ver
// push_expo.go/enviarPushExpoATodos en el backend).
//
// Mientras la app se pruebe con Expo Go, este flujo puede no completarse
// del todo (Expo dejó de dar soporte a push remoto dentro de Expo Go en
// Android; iOS tiene sus propias limitaciones ahí también) -- funciona
// completo en un development build o en el build final de la tienda
// (`eas build`), ver API_MOVIL.md. Por eso todo aquí está escrito para
// fallar en silencio (con aviso en consola) en vez de tronar la app si el
// token no se puede obtener.

// Controla cómo se ve una notificación mientras la app está ABIERTA en
// primer plano -- sin esto, por defecto no se muestra nada visible.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

// configurarCanalAndroid -- Android 8+ exige un "canal" antes de poder
// mostrar notificaciones; sin esto llegan sin sonido en algunos equipos.
export async function configurarCanalAndroid() {
  if (Platform.OS !== 'android') return;
  await Notifications.setNotificationChannelAsync('default', {
    name: 'Pasitos',
    importance: Notifications.AndroidImportance.DEFAULT,
    vibrationPattern: [0, 200, 200, 200],
  });
}

// obtenerTokenExpo -- pide permiso (si hace falta) y regresa el token de
// push de este dispositivo, o null si no se pudo (simulador/emulador sin
// Google Play, permiso negado, o falta el projectId de EAS -- ver el
// comentario de arriba).
export async function obtenerTokenExpo() {
  if (!Device.isDevice) {
    console.log('Push: los simuladores/emuladores no reciben push de verdad, se omite.');
    return null;
  }

  const { status: estadoActual } = await Notifications.getPermissionsAsync();
  let estadoFinal = estadoActual;
  if (estadoActual !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    estadoFinal = status;
  }
  if (estadoFinal !== 'granted') {
    console.log('Push: permiso de notificaciones no concedido.');
    return null;
  }

  await configurarCanalAndroid();

  const projectId = Constants.expoConfig?.extra?.eas?.projectId ?? Constants.easConfig?.projectId;
  if (!projectId) {
    console.log('Push: falta el projectId de EAS (corre "eas init" primero) -- no se puede pedir el token.');
    return null;
  }

  try {
    const { data: token } = await Notifications.getExpoPushTokenAsync({ projectId });
    return token;
  } catch (err) {
    console.log('Push: no se pudo obtener el token de Expo:', err.message);
    return null;
  }
}

// activarPushNativo -- obtiene el token y lo registra en el backend. Regresa
// el token si quedó activo, o null si no (mismo criterio de "fallar en
// silencio" de arriba).
export async function activarPushNativo() {
  const token = await obtenerTokenExpo();
  if (!token) return null;
  await api.post('/push/expo/registrar', { token });
  return token;
}

// notificacionesActivas -- true si el sistema operativo ya tiene el permiso
// concedido, para pintar el interruptor en el estado correcto al abrir la
// app (sin volver a pedir permiso solo por consultarlo).
export async function notificacionesActivas() {
  if (!Device.isDevice) return false;
  const { status } = await Notifications.getPermissionsAsync();
  return status === 'granted';
}

// desactivarPushNativo -- da de baja el token actual del dispositivo en el
// backend (se llama al cerrar sesión o al apagar el interruptor de
// notificaciones).
export async function desactivarPushNativo() {
  const token = await obtenerTokenExpo();
  if (!token) return;
  try {
    await api.delete('/push/expo/registrar', { data: { token } });
  } catch (err) {
    console.log('Push: no se pudo eliminar el token:', err.message);
  }
}
