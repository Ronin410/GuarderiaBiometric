# App móvil de Pasitos (React Native / Expo) — para papás

Una sola app en React Native para Android e iOS, como pediste ("no quiero
dos aplicaciones diferentes"). Usa [Expo](https://expo.dev) en vez de React
Native "a pelo" por una razón muy concreta para este proyecto: se puede
**probar en tu propio celular en minutos, sin instalar Android Studio ni
Xcode**, y más adelante generar el instalable real (APK / IPA) **en la nube
de Expo**, sin necesitar una Mac para iOS -- eso último era justo el
obstáculo que teníamos con el intento anterior (Capacitor).

**Alcance de esta primera pasada, según lo platicado: solo papás.**
Personal/admin sigue usándose por navegador, como hasta ahora.

## Qué ya funciona

- **Login** (`src/screens/LoginScreen.js`) -- correo/contraseña contra el
  mismo backend de siempre (`POST /login`, `tipo: "papa"`).
- **Inicio** (`src/screens/DashboardScreen.js`) -- saludo, aviso de "las
  bitácoras se actualizan en tiempo real", accesos a Chat/Encuestas/
  Eventos/Menú semanal (por ahora "Próximamente", ver abajo) y el listado
  de niños.
- **Bitácora de hoy** (`src/screens/BitacoraScreen.js`) -- entrada/salida,
  alimentación, siesta, esfínter, fotos del día (con vista ampliada) y la
  nota de la maestra, navegable día por día (igual que la pestaña "Hoy" de
  `VistaPadreDetalle.jsx` en la web). Es la función más importante de la
  app, así que fue la primera en portarse completa.
- **Chat con la guardería** (`src/screens/ChatContactosScreen.js` +
  `ChatHiloScreen.js`) -- selector de con quién hablar (personal/admin) y
  el hilo de mensajes: texto, adjuntos (imagen o PDF, hasta 10 MB, con
  vista ampliada de fotos), separadores "Hoy"/"Ayer"/fecha, y
  actualización cada 5 segundos (mismo polling que ya usa la web -- no hay
  push en tiempo real todavía, ver "Lo que sigue").
- **Encuestas** (`src/screens/EncuestasScreen.js`) -- opción múltiple y
  texto libre, igual que `EncuestasPadre.jsx` de la web: el backend solo
  deja responder una vez (rechaza un segundo envío con 400), y ya
  respondida se ve el mismo formulario pero deshabilitado con lo que el
  papá puso, en vez de ocultarlo.
- **Sesión**: igual que en la web, la cookie con el JWT es httpOnly y la
  guarda automáticamente el cliente HTTP nativo (no hay que leerla ni
  gestionarla a mano). El token CSRF viaja en memoria, igual que en
  `axiosConfig.js` de la web (`src/api/client.js` es el equivalente aquí).

## Diferencia importante con el intento de Capacitor: no hace falta tocar CORS

Con Capacitor, la app corría dentro de un WebView que se comporta como un
navegador (manda encabezado `Origin`, por eso había que agregar
`https://localhost` a `ALLOWED_ORIGINS` en el backend). **Esta app no usa
WebView** -- las peticiones (axios) las hace directo el cliente HTTP nativo
de iOS/Android, que normalmente **no manda encabezado `Origin`** en
absoluto (es una noción exclusiva de navegadores). Sin ese encabezado, el
middleware de CORS del backend (`gin-contrib/cors`) ni siquiera evalúa la
petición como "cross-origin" -- debería funcionar contra el backend de
producción tal cual está, sin tocar `ALLOWED_ORIGINS`. Confírmalo como
primera prueba al correrla (intenta iniciar sesión); si por algún motivo
fallara por CORS, avísame y lo ajustamos.

## Cómo probarla AHORA MISMO en tu celular (sin Android Studio ni Xcode)

Esto se corre en tu computadora, no desde este entorno remoto (aquí no hay
forma de que tu celular vea este contenedor). Ya con la rama en tu máquina:

```bash
cd GuarderiaBiometricMobile
npm install
npx expo start
```

Te va a salir un código QR en la terminal:

- **Android**: instala la app **Expo Go** (Play Store) y escanéalo desde ahí.
- **iOS**: instala **Expo Go** (App Store) y escanéalo con la cámara normal
  del iPhone (te ofrece abrirlo en Expo Go).

La app carga directo en tu celular, con hot-reload -- cualquier cambio que
haga al código se refleja ahí casi al instante, sin recompilar nada nativo.

## Cómo generar el instalable real más adelante (APK / IPA), sin Mac

Cuando ya quieras un instalable de verdad para el piloto (no solo Expo Go):

```bash
npm install -g eas-cli
eas login          # cuenta gratuita de Expo
eas build --platform android --profile preview
eas build --platform ios --profile preview
```

Esto compila en los servidores de Expo (incluye una Mac en la nube para
iOS) y te da un link para descargar el APK/IPA -- se instala directo en el
celular del papá de prueba, o se sube a TestFlight, sin que tú necesites
Android Studio ni una Mac física. Lo dejamos para cuando ya estés
conforme con las pantallas, para no gastar builds de más mientras se sigue
ajustando la app.

## Lo que sigue (orden sugerido)

1. **Circulares/Eventos** -- listas de solo lectura, las más sencillas que
   quedan.
2. **Menú semanal** -- ya existe el endpoint (`/padre/menu-semanal`), solo
   falta la pantalla.
3. El resto de pestañas de `VistaPadreDetalle.jsx` (Expediente, Pagos,
   Ausencias, Comedor, Galería) dentro de la pantalla de bitácora de cada
   niño.
4. **Notificaciones push nativas** (Expo Notifications, que por debajo usa
   FCM/APNs) -- las que ya existen hoy son Web Push, pensadas para la PWA
   del navegador; para esta app hace falta el equivalente nativo, que
   Expo también simplifica bastante frente a hacerlo a mano. Esto además
   reemplazaría el polling cada 5 segundos del chat por avisos reales al
   instante.

## Estructura del proyecto

```
GuarderiaBiometricMobile/
  App.js                        -- navegación raíz (login vs. app)
  src/
    theme.js                    -- mismos colores de marca que la web
    api/client.js                -- axios + manejo de CSRF (igual que axiosConfig.js)
    context/AuthContext.js       -- sesión: /me, /login, /logout
    utils/fecha.js                -- helpers de fecha (America/Mazatlan)
    screens/
      LoginScreen.js
      DashboardScreen.js
      BitacoraScreen.js
      ChatContactosScreen.js      -- selector de con quién hablar
      ChatHiloScreen.js           -- hilo de mensajes con un contacto
      EncuestasScreen.js          -- responder encuestas (una sola vez)
      ProximamenteScreen.js      -- placeholder para lo que falta portar
```
