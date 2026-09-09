# Estado Actual del Proyecto — GuarderiaBiometric

> Documento generado a partir de un escaneo completo del repositorio (backend + frontend) el 2026-07-20.

## 1. Visión general

Sistema de control de asistencia biométrico para guarderías (multi-sede/"multi-tenant" por `guarderia_id`). Permite:

- Registrar el rostro de un tutor/padre mediante cámara web y vincularlo a AWS Rekognition.
- Identificar al tutor por su rostro en la entrada/salida y registrar asistencia de sus hijos.
- Llevar una "bitácora diaria" pedagógica por niño (alimentación, siesta, esfínter, fotos, observaciones).
- Compartir esa bitácora con el tutor vía un enlace público (WhatsApp) sin necesidad de login.
- Generar reportes de asistencia filtrables e imprimibles para el personal/administración.
- Un dashboard exclusivo para el rol "papá/tutor" donde ve a sus hijos y el reporte diario de cada uno.

Es una aplicación de una sola sede lógica por "guardería", con datos segmentados por `guarderia_id` en casi todas las consultas (multi-tenant a nivel de fila).

## 2. Stack tecnológico

### Backend (`GuarderiaBiometricBack/`)
- **Lenguaje/Framework**: Go 1.24 + [Gin](https://github.com/gin-gonic/gin).
- **Base de datos**: PostgreSQL, accedida vía `database/sql` + `github.com/lib/pq`. Dos conexiones separadas: `db` (datos operativos) y `dbAuth` (usuarios/autenticación), configuradas por `DATABASE_URL` y `DATABASE_URL_AUTH`.
- **Reconocimiento facial**: AWS Rekognition (`SearchFacesByImage`, `IndexFaces`) mediante AWS SDK v2.
- **Almacenamiento de fotos**: AWS S3 (bucket `biosafe-storage-fotos`), subida directa desde el backend con ACL público de lectura.
- **Autenticación**: JWT (`golang-jwt/jwt/v5`) con expiración de 24h, más un PIN numérico de 4 dígitos (`pin_admin`) como segundo factor para desbloquear pestañas administrativas.
- **Contraseñas**: hash con `bcrypt` (costo 10).
- **Tareas programadas**: `robfig/cron/v3` — cierre automático nocturno (23:00 hora `America/Mazatlan`) que genera una "SALIDA" para todo niño que quedó con "ENTRADA" abierta.
- **CORS**: abierto a cualquier origen (`AllowOrigins: []string{"*"}`) con credenciales habilitadas.
- **Migraciones**: se ejecutan automáticamente al iniciar (`RunMigrations()`), con `CREATE TABLE IF NOT EXISTS`.
- Todo el backend vive en **un único archivo `main.go`** (~1600 líneas): tipos, middleware, rutas y migraciones juntos. Las carpetas `BD/`, `Comandos/` y `Rostrosprueba/` existen pero están vacías en el árbol versionado (probablemente ignoradas o usadas solo localmente).

### Frontend (`GuarderiaBiometricFront/`)
- **Framework**: React 19 + Vite 7.
- **Estilos**: Tailwind CSS 4 (sin `theme.extend`, colores usados como clases utilitarias literales).
- **Enrutamiento**: `react-router-dom` v7, pero usado de forma mínima (ver sección 4).
- **HTTP**: `axios` con una instancia central (`axiosConfig.js`) + interceptores.
- **Cámara**: `react-webcam` para la captura biométrica.
- **UI/UX auxiliares**: `lucide-react` (iconos), `sweetalert2` + `sweetalert2-react-content` (usado solo en una vista).
- **PWA**: `vite-plugin-pwa` + manifest estático (`public/manifest.json`) + registro manual de Service Worker en `index.html`.
- **Dev server**: HTTPS local forzado (`@vitejs/plugin-basic-ssl`) porque el acceso a la cámara requiere contexto seguro.
- **Despliegue backend**: apunta a `https://guarderiabiometricback.onrender.com` (Render), hardcodeado en el frontend.

## 3. Modelo de datos (PostgreSQL)

| Tabla | Propósito | Campos clave |
|---|---|---|
| `guarderias` | Sedes/tenants | `id`, `nombre`, `slug` (único), `direccion`, `plan_suscripcion` |
| `usuarios` | Cuentas de personal | `id`, `guarderia_id`, `username` (único), `password_hash`, `pin_admin`, `rol` (`admin`/`staff`) |
| `hijos` | Niños | `id`, `nombre_niño`, `guarderia_id`, `activo`, `url_token` (UUID para el reporte público) |
| `padres` | Tutores registrados biométricamente | `id`, `nombre`, `face_id` (único, de Rekognition), `guarderia_id`, `celular`, `recibe_whatsapp` |
| `tutor_hijos` | Relación N:M tutor↔niño | `padre_id`, `hijo_id`, `guarderia_id` (PK compuesta) |
| `asistencia` | Movimientos de entrada/salida | `padre_id`, `hijo_id`, `guarderia_id`, `fecha_hora`, `aseado`, `reporte_golpe`, `observaciones`, `tipo_movimiento` (`ENTRADA`/`SALIDA`/`REGISTRO`) |
| `seguimiento_diario` | Bitácora pedagógica diaria | `hijo_id`, `guarderia_id`, `fecha`, `desayuno`, `comida`, `merienda`, `esfinter`, `durmio`, `observaciones` — único por `(hijo_id, fecha)` |
| `fotos_seguimiento` | Fotos ligadas a una bitácora | `seguimiento_id`, `url` (S3) |

Índice adicional: `idx_asistencia_fecha` sobre `asistencia(fecha_hora DESC)`.

Nota: `padres.celular` y `padres.recibe_whatsapp` se usan en queries (`/seguimiento`) pero **no aparecen en `RunMigrations()`**, por lo que deben haberse agregado manualmente en la base de datos de producción fuera del código versionado.

## 4. Endpoints del backend (todos bajo `main.go`)

### Públicos (sin JWT)
| Método | Ruta | Función |
|---|---|---|
| POST | `/usuarios/registro` | Crear usuario de personal |
| POST | `/login` | Login (usuario/contraseña) → JWT |
| GET | `/publico/seguimiento/:token` | Bitácora diaria de un niño vía token UUID (para compartir por WhatsApp) |

### Protegidos (requieren `Authorization: Bearer <JWT>`)
| Método | Ruta | Función |
|---|---|---|
| POST | `/registrar` | Registra rostro de un tutor en Rekognition + crea `padre` |
| POST | `/identificar` | Identifica rostro, devuelve tutor + hijos + último estado |
| POST | `/registrar-hijo` | Crea un niño |
| GET | `/padre/:id/hijos` | Hijos de un tutor (con comodín `:id=0` → usa el ID del token) |
| POST | `/confirmar-asistencia` | Registra ENTRADA/SALIDA (alterna automáticamente) |
| POST | `/vincular-tutor` | Vincula tutor↔niño |
| GET | `/buscar-hijos?q=` | Búsqueda de niños por nombre |
| GET | `/buscar-padres?q=` | Búsqueda/listado de tutores con sus hijos |
| POST | `/desvincular-hijo` | Elimina relación tutor↔niño |
| POST | `/actualizar-padre` | Renombra un tutor |
| GET | `/bitacora?fecha=` | Roster diario de asistencia + bitácora resumida |
| GET | `/reportes-asistencia?inicio=&fin=` | Reporte detallado por rango de fechas |
| POST | `/verificar-pin` | Valida el PIN admin de 4 dígitos |
| PATCH | `/hijos/:id/desactivar` / `/activar` | Baja/alta lógica de un niño |
| PUT | `/hijos/:id` | Renombra un niño |
| POST | `/admin/forzar-estatus` | Fuerza ENTRADA/SALIDA manualmente (sin escaneo facial) |
| POST | `/seguimiento` (multipart) | Crea/actualiza bitácora diaria + sube fotos a S3 |
| GET | `/seguimiento/:hijo_id?fecha=` | Bitácora diaria de un niño (vista autenticada) |

Toda la lógica multi-tenant se implementa filtrando por `guarderia_id` (tomado del JWT) en cada query. Rekognition usa una `CollectionId` calculada como `"guarderia-<guarderia_id>"` en el backend — aunque el frontend actualmente envía el literal `"guarderia-rostros"` (ver documento de mejoras).

## 5. Frontend — estructura y componentes

Enrutamiento real (`react-router-dom`): solo existen dos rutas — `/seguimiento/:token` (reporte público) y un catch-all `/*` que renderiza toda la app autenticada. La navegación interna del panel de personal (kiosco, admin, bitácora, reportes) **no usa el router**, es manejada con un `useState('tab')` local, por lo que no es enlazable por URL ni conserva estado al refrescar.

| Archivo | Rol |
|---|---|
| `App.jsx` | Router raíz + componente `MainApp` (login, kiosco de escaneo facial, pestañas admin/bitácora/reportes, modal de PIN) |
| `axiosConfig.js` | Instancia axios central con interceptores de auth (agrega `Bearer token`) y manejo de 401 |
| `DashboardPadre.jsx` | Landing del rol "papá": lista de hijos y acceso a su detalle |
| `VistaPadreDetalle.jsx` | Detalle de bitácora diaria de un hijo (vista autenticada del tutor) |
| `ReportePublico.jsx` | Misma bitácora diaria pero vía token público, sin login |
| `FormularioBitacora.jsx` | Formulario que llena el personal (comida, siesta, esfínter, fotos) + genera enlace de WhatsApp |
| `VistaBitacora.jsx` | Roster diario del personal con estado de cada niño y acceso al formulario de bitácora |
| `GestionHijos.jsx` | Panel admin: vincular/desvincular niños, editar nombres, dar de alta/baja |
| `PanelReportes.jsx` | Tabla de reportes de asistencia filtrable, ordenable e imprimible |

**Autenticación en frontend**: JWT + `role`, `userId`, `guarderia_nombre`, `guarderia_slug` guardados como claves sueltas en `localStorage`. El interceptor de axios adjunta el token a cada request y, ante un 401, limpia solo el `token` y redirige a `/login` (ruta que no existe explícitamente, cae al catch-all).

**Control de acceso**: es binario — rol `papa` ve el dashboard de padres; cualquier otro rol ve el kiosco de personal. Dentro del kiosco, el acceso a pestañas administrativas (`admin`, `bitácora`, `reportes`) se protege con un PIN compartido, no con permisos por usuario/rol.

**Flujo biométrico**: `react-webcam` captura una imagen (`getScreenshot()`), se envía como base64 a `/registrar` o `/identificar` junto con un `collection_id` fijo. No hay validación de "hay un rostro en cuadro" del lado del cliente.

**PWA**: configurada por partida triple — `vite-plugin-pwa` (manifest inline + auto-registro de SW), un `public/manifest.json` estático enlazado explícitamente en `index.html`, y un registro manual de `/sw.js` en un `<script>` propio.

## 6. Despliegue e infraestructura

- Backend desplegado en **Render** (`guarderiabiometricback.onrender.com`), URL hardcodeada en dos archivos del frontend.
- Credenciales AWS (Rekognition + S3) tomadas de variables de entorno (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`); el backend aborta el arranque si faltan.
- Cadenas de conexión a Postgres vía `DATABASE_URL` y `DATABASE_URL_AUTH`.
- No hay pipeline de CI/CD, ni archivo de configuración de despliegue (Dockerfile, render.yaml, etc.) versionado en el repo.
- No hay pruebas automatizadas (unitarias, integración ni E2E) en ninguno de los dos proyectos.

## 7. Historial de control de versiones

El repositorio tiene un único commit ("Se une proyecto de guarderia biometrica"), es decir, se integró como snapshot único sin historial de desarrollo previo.
