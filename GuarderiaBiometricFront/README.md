# Pasitos — Frontend

App web (React + Vite, instalable como PWA) para el sistema biométrico de guarderías: kiosco de identificación facial, bitácora diaria, panel de administración (Perfiles, Pagos, Estadísticas) y portal para los papás con notificaciones push.

## Requisitos

- Node.js 20+
- El backend (`GuarderiaBiometricBack/`) corriendo y accesible — ver su propio README.

## Configuración

Copia `.env.example` a `.env.local` y ajusta:

- `VITE_API_URL` — URL del backend (ej. `https://localhost:8099` en local, la URL de producción en despliegue).
- `VITE_VAPID_PUBLIC_KEY` — clave pública VAPID para notificaciones push (debe coincidir con `VAPID_PUBLIC_KEY` del backend). Si se omite, la app funciona igual pero el botón "Activar notificaciones" del portal del papá no hace nada.

## Correr localmente

```bash
npm install
npm run dev
```

El servidor de desarrollo usa **HTTPS con certificado autofirmado** (`@vitejs/plugin-basic-ssl`) porque el acceso a la cámara del kiosco requiere un contexto seguro — el navegador mostrará una advertencia de certificado la primera vez, es normal en local.

Otros scripts:

```bash
npm run build    # build de producción a dist/
npm run preview  # sirve el build de producción localmente
npm run lint     # ESLint
```

## Estructura

| Ruta | Qué es |
|---|---|
| `src/App.jsx` | Login, kiosco biométrico y todas las pestañas del personal (Familia, Bitácora, Reportes, Perfiles, Pagos, Estadísticas) |
| `src/DashboardPadre.jsx` / `VistaPadreDetalle.jsx` | Portal del papá (bitácora del día, expediente, pagos, notificaciones) |
| `src/PanelPerfiles.jsx`, `PanelPagos.jsx`, `PanelEstadisticas.jsx` | Módulo de Administración |
| `src/GestionHijos.jsx`, `FormularioBitacora.jsx`, `VistaBitacora.jsx`, `PanelReportes.jsx` | Gestión de tutores/niños, bitácora diaria y reportes clásicos |
| `src/ReportePublico.jsx` | Vista pública sin login (enlace compartido por WhatsApp) |
| `src/axiosConfig.js` | Cliente HTTP central (agrega el token, maneja 401) |
| `src/utils/` | `fecha.js` (fecha local unificada), `alertas.js` (diálogos con SweetAlert2), `push.js` (suscripción a notificaciones) |
| `src/sw.js` | Service Worker propio (precache + notificaciones push) |

## PWA y notificaciones push

El Service Worker se genera con `vite-plugin-pwa` en modo `injectManifest`, a partir de `src/sw.js` — así se puede tener el precache normal de una PWA **y además** escuchar el evento `push` para mostrar notificaciones. En desarrollo (`npm run dev`), el Service Worker **no se registra** por defecto (comportamiento estándar de `vite-plugin-pwa`), así que el botón de notificaciones fallará con un aviso de "Service Worker no activo" — esto es esperado; para probarlo de verdad hay que correr `npm run build && npm run preview`.

## Roles y pantallas

- **admin**: acceso completo sin PIN.
- **staff**: Kiosco y Registro libres; Familia/Bitácora/Reportes/Perfiles/Pagos/Estadísticas piden el PIN configurado por la guardería.
- **papa**: pantalla completamente distinta (`DashboardPadre`), solo ve a sus propios hijos.
