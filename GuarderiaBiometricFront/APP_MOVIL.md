# App móvil de Pasitos (Android / iOS) — para papás

Arranque de la versión instalable de Pasitos para Android e iOS, usando
[Capacitor](https://capacitorjs.com/) para empacar la misma web app que ya
funciona (React/Vite) dentro de una app nativa real -- instalable desde un
ícono en el celular, sin abrir el navegador.

**Alcance de esta primera pasada (a propósito, según lo platicado):** que la
experiencia de los **papás** sea intuitiva e instalable. Personal/admin
sigue usándose por navegador normal, como hasta ahora -- no se tocó nada de
esa parte.

## Qué se hizo

- **Capacitor** agregado al proyecto (`capacitor.config.json`), con las
  carpetas nativas `android/` y `ios/` ya generadas y comiteadas (así es
  como Capacitor espera que se maneje: son proyectos reales de Android
  Studio / Xcode, no se regeneran solos).
- **La app abre directo en el login de papás**, no en la página de
  presentación (`LandingPage`) -- ver `EntradaRaiz` en `App.jsx`. Tiene
  sentido: quien instaló la app ya sabe qué es Pasitos, no necesita que se
  la vuelvan a vender. El toggle de acceso también arranca en "Papá" en vez
  de "Personal" dentro de la app nativa (en la web se queda igual).
- **Cámara para el reconocimiento facial**: el permiso de Android está
  declarado en `AndroidManifest.xml`, `Info.plist` trae el texto que iOS
  exige (`NSCameraUsageDescription`), y `main.jsx` pide el permiso en
  tiempo de ejecución apenas arranca la app (`solicitarPermisoCamaraNativo`
  en `src/utils/nativeApp.js`) para que ya esté resuelto cuando el papá
  llegue a identificarse/enrolar.
- **Orientación fija vertical** en ambas plataformas -- el diseño de las
  pantallas de papás es de tarjetas centradas pensadas para celular en
  vertical, igual que ya declaraba `manifest.json` de la PWA.
- **Ícono y splash screen** generados a partir del logo de Pasitos
  (`assets/icon.png`, `assets/splash.png` -- fuente para regenerarlos con
  `npx capacitor-assets generate --android --ios` si el logo cambia; **no**
  agregar `--pwa`, eso pisa `public/manifest.json` de la PWA que no
  queremos tocar).
- **Nada de esto toca la PWA/web actual** en Render: sigue funcionando
  exactamente igual que antes para quien entra por navegador.

## Antes de instalar en el celular de la guardería de prueba: CORS

El backend solo acepta peticiones (login, API, todo) de los orígenes que
tenga en la variable de entorno `ALLOWED_ORIGINS` (Render → el servicio del
backend → Environment). La app empacada se presenta ante el backend como el
origen **`https://localhost`** (se forzó así en `capacitor.config.json`,
igual en Android que en iOS, justo para que las cookies de sesión
`SameSite=None; Secure` que ya usa el login funcionen igual que en la web).

**Hay que agregar `https://localhost` a `ALLOWED_ORIGINS` en el backend**
(separado por coma de los orígenes que ya estén, sin quitar ninguno) o el
login desde la app nativa fallará por CORS. Esto es un cambio de
configuración en Render, no de código -- avísame cuando quieras que lo haga
o hazlo tú directo en el dashboard.

## Cómo generar el instalable para el piloto

### Android (APK directo, sin pasar por Play Store)

Esto necesita **Android Studio** (o al menos el SDK de Android + un JDK) en
la máquina donde se corra -- este entorno de trabajo remoto no tiene acceso
a los servidores de Google para descargarlo, así que el build real hay que
hacerlo en tu laptop o en un CI con internet normal.

```bash
cd GuarderiaBiometricFront
npm install          # primera vez
npm run cap:android:apk
```

El APK queda en `android/app/build/outputs/apk/debug/app-debug.apk`.
Se lo mandas al papá de prueba (WhatsApp, Drive, USB, lo que sea) y en su
Android: **Ajustes → Seguridad → permitir instalar apps de "orígenes
desconocidos"** para esa fuente, y lo instala tocando el archivo. No pasa
por revisión de Google Play -- es exactamente lo que pediste para probar
ahorita.

Si prefieres verlo/tocarlo en Android Studio en vez de la terminal:
`npm run cap:android` (compila y lo abre directo en Android Studio).

### iOS (TestFlight)

Esto **sí necesita una Mac con Xcode** y una cuenta de Apple Developer
($99 USD/año) -- no hay forma de generarlo desde Linux, ni desde este
entorno remoto. Cuando tengas la Mac a mano:

```bash
cd GuarderiaBiometricFront
npm run cap:ios       # compila y abre el proyecto en Xcode
```

Desde Xcode: seleccionar tu Team de Apple Developer, Product → Archive, y
subirlo a App Store Connect para distribuirlo por TestFlight a los papás
de prueba (instalan la app TestFlight y ahí les llega, sin pasar por
revisión completa de la App Store todavía).

## Después de cualquier cambio en el código de React

Los archivos nativos (`android/`, `ios/`) no se actualizan solos -- hay que
volver a compilar la web y copiarla:

```bash
npm run cap:sync
```

(`npm run cap:android` / `npm run cap:ios` ya lo hacen antes de abrir el
proyecto nativo, no hace falta correrlo aparte si vas a abrir uno de esos).

## Lo que se dejó fuera a propósito, para después

- **Notificaciones push nativas** (Firebase Cloud Messaging en Android /
  APNs en iOS). Las notificaciones push que ya existen (entradas, salidas,
  chat de soporte) usan Web Push -- funcionan en la PWA del navegador, pero
  dentro del WebView de una app empacada no son confiables de la misma
  forma. Es un proyecto aparte (dar de alta un proyecto de Firebase,
  `@capacitor/push-notifications`, tocar `push.go` para mandar también por
  FCM/APNs) -- lo dejamos para cuando ya esté rodando esto primero.
- **Vista nativa para personal/staff** -- por ahora la app instalada es
  para papás; el personal sigue entrando por navegador como siempre.
- **Publicación real en Play Store / App Store** -- esta primera vuelta es
  para instalar directo en el piloto, sin pasar por revisión de las
  tiendas.
