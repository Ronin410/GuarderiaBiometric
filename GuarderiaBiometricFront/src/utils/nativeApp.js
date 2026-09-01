import { Capacitor } from '@capacitor/core';
import { Camera } from '@capacitor/camera';

// Puente mínimo con la app nativa (Capacitor) -- todo lo de aquí es un
// no-op cuando se corre como web/PWA normal (Capacitor.isNativePlatform()
// da false ahí), así que es seguro llamarlo siempre desde main.jsx sin
// tener que ramificar el resto de la app por plataforma.

// solicitarPermisoCamaraNativo -- en la app empacada (Android/iOS) el
// <video> de react-webcam necesita el permiso de cámara del SISTEMA
// OPERATIVO antes de que el WebView se lo pueda conceder al getUserMedia()
// de la página (el <uses-permission> de AndroidManifest.xml solo habilita
// que la app PUEDA pedirlo, no lo concede). Se pide una sola vez al
// arrancar la app para que, cuando el papá llegue a identificar/enrolar
// rostro, ya esté resuelto -- sin esto la cámara del reconocimiento facial
// se queda en negro sin ningún aviso.
export async function solicitarPermisoCamaraNativo() {
  if (!Capacitor.isNativePlatform()) return;
  try {
    await Camera.requestPermissions({ permissions: ['camera'] });
  } catch (err) {
    console.error('No se pudo solicitar el permiso de cámara nativo:', err);
  }
}
