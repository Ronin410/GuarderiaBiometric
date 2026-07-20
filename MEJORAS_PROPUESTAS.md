# Mejoras Propuestas — GuarderiaBiometric

> Basado en el escaneo completo del proyecto (ver `ESTADO_ACTUAL.md`). Ordenado por impacto/riesgo, no por esfuerzo.

## Estado de implementación (2026-07-20)

Ya implementados en esta rama:
- Seguridad: `JWT_SECRET` por variable de entorno, eliminado el log de contraseñas en texto plano (y el `pin_admin` ya no se devuelve en `/login`), CORS restringido vía `ALLOWED_ORIGINS`, rate limiting en `/login`, `/identificar` y `/verificar-pin`.
- Bugs: typo `'2d-digit'`, cálculo de fecha "hoy" unificado (`src/utils/fecha.js`), interceptor 401 ahora limpia todo `localStorage`, guarda null-safe en `PanelReportes.jsx`, props no usados corregidos (`DashboardPadre`, `GestionHijos`), y dos bugs adicionales encontrados durante la implementación: el login nunca devolvía `user_id` (por eso `padreId` llegaba vacío) y la verificación de PIN comparaba contra un texto que el backend nunca envía (el modal de PIN nunca desbloqueaba nada).
- Arquitectura: URL del backend centralizada vía `VITE_API_URL` (`.env.example` en ambos proyectos), configuración PWA deduplicada (una sola fuente de manifest y un solo registro de Service Worker).

Pendiente (no abordado en esta pasada, ver detalle en cada sección): separar `main.go` en paquetes, pruebas automatizadas, migrar pestañas internas a `react-router`, deduplicar `ReportePublico`/`VistaPadreDetalle`, herramienta de migraciones dedicada, CI/CD, monitoreo, revisión de accesibilidad.

## 1. Seguridad (prioridad alta)

- **Clave JWT hardcodeada en el código fuente**: `jwtKey = []byte("tu_clave_secreta_super_segura")` en `main.go`. Debe moverse a una variable de entorno (`JWT_SECRET`) y rotarse, ya que al estar en el historial de git cualquiera con acceso al repo puede falsificar tokens.
- **CORS totalmente abierto**: `AllowOrigins: []string{"*"}` combinado con `AllowCredentials: true` es una combinación insegura (y en la práctica varios navegadores la rechazan). Restringir a los dominios reales del frontend (producción + entornos de prueba).
- **PIN admin de 4 dígitos como único control de acceso a paneles sensibles**: `/verificar-pin` no tiene límite de intentos (rate limiting/backoff), por lo que es vulnerable a fuerza bruta (solo 10,000 combinaciones). Además, el PIN se devuelve en texto plano en la respuesta del `/login` (`"pin_admin": pin`) y se guarda en `localStorage` del lado del cliente en algunos flujos — debería no exponerse nunca al frontend fuera del momento de verificación.
- **JWT almacenado en `localStorage`**: expone el token a robo vía XSS. Si el riesgo de XSS es relevante para este proyecto, considerar cookies `httpOnly` + `SameSite`. Como mínimo, sanear cualquier input renderizado sin escapar.
- **Sin manejo de expiración/refresh de token**: el usuario solo descubre que su sesión expiró cuando una petición falla con 401. Agregar refresh silencioso o aviso proactivo antes de que expire (el JWT dura 24h).
- **Logs de credenciales en texto plano**: `main.go` imprime `fmt.Printf("Intentando login plano: Usuario[%s] Pass[%s]\n", ...)` — esto escribe la contraseña del usuario en los logs del servidor. Eliminar de inmediato; es una fuga de credenciales en cualquier sistema de logging centralizado.
- **Endpoint público de bitácora protegido solo por "URL secreta"**: `/publico/seguimiento/:token` no tiene expiración, límite de uso ni posibilidad de revocación del `url_token`. Si el enlace de WhatsApp se reenvía o queda indexado, cualquiera con el link accede indefinidamente a fotos y datos del niño. Considerar expiración temporal (p. ej. válido por 7 días) o invalidación bajo demanda.
- **Falta de rate limiting general** en endpoints sensibles (`/login`, `/identificar`, `/verificar-pin`) — abre la puerta a ataques de fuerza bruta y abuso de costos de Rekognition (cada llamada a `/identificar` cuesta dinero en AWS).
- **CollectionId de Rekognition inconsistente entre frontend y backend**: el backend calcula `guarderia-<id>` pero el frontend envía el literal `"guarderia-rostros"` en `App.jsx`. Si esto es realmente ignorado por el backend (porque el backend recalcula el ID desde el JWT) no hay bug funcional, pero conviene eliminar el campo del payload del frontend para no sugerir una falsa fuente de verdad y evitar confusión futura.

## 2. Correctitud / bugs concretos

- **`DashboardPadre.jsx` llama a `GET /padre/0/hijos`** con un `0` literal en lugar de usar el ID del padre autenticado. Aunque el backend ya resuelve el comodín `0` con el ID del token, el prop `padreId` que le pasa `App.jsx` queda sin usar — limpiar la firma del componente o usarla explícitamente para evitar futuras confusiones si cambia la lógica del backend.
- **`usuarioNombre` en `DashboardPadre.jsx` siempre cae al valor por defecto `"Familia"`** porque lee `localStorage.getItem('username')`, pero esa clave nunca se guarda en el login (`App.jsx` solo guarda `token`, `role`, `userId`, `guarderia_nombre`, `guarderia_slug`). Agregar `localStorage.setItem('username', ...)` en el login o pasar el username real desde el backend.
- **Props declarados pero nunca usados** (indican refactors incompletos): `DashboardPadre` ignora `padreId` y `alCerrarSesion`; `FormularioBitacora` no usa `datosEntrada`; `GestionHijos` declara `onFinalizar` pero nunca lo invoca. Revisar cada caso y decidir si se elimina el prop o se completa su uso.
- **Typo en `VistaBitacora.jsx`**: `{ hour: '2d-digit', minute: '2d-digit', hour12: true }` — el valor válido es `'2-digit'`, no `'2d-digit'`. El navegador ignora la opción inválida silenciosamente, así que el formato de hora no se controla como se pretende.
- **Interceptor de 401 en `axiosConfig.js` limpia solo `token`**, dejando `role`, `userId`, `guarderia_nombre` y `guarderia_slug` obsoletos en `localStorage`, a diferencia del logout explícito que sí hace `localStorage.clear()`. Puede producir estados inconsistentes tras un 401 (p. ej. la UI cree que sigue en una guardería que ya no aplica).
- **`PanelReportes.jsx` puede lanzar una excepción** si el backend devuelve un registro con `hijo_nombre` nulo (`reg.hijo_nombre.toLowerCase()` sin guardas), rompiendo el filtro/orden completo de la tabla.
- **`ReportePublico.jsx` y `VistaPadreDetalle.jsx` calculan la fecha "de hoy" con `toISOString().split('T')[0]` (UTC)**, mientras `VistaBitacora.jsx` calcula lo mismo corrigiendo manualmente a la zona horaria de Culiacán y `PanelReportes.jsx` usa `toLocaleDateString('en-CA')`. Cerca de medianoche, un usuario puede ver la fecha equivocada dependiendo de qué componente esté usando. Unificar en una sola función de utilidad de fecha local.

## 3. Arquitectura y mantenibilidad

- **`main.go` concentra ~1600 líneas** con tipos, middleware, ~20 rutas y migraciones SQL en un solo archivo. Separar en paquetes (`handlers`, `models`, `middleware`, `db/migrations`) mejoraría legibilidad y facilitaría pruebas unitarias.
- **Backend sin capa de pruebas**: no hay ni un solo test (`_test.go`) en el proyecto Go ni en el frontend. Priorizar pruebas de los endpoints críticos (login, identificación facial, confirmación de asistencia) y de la lógica de fecha/zona horaria, que es propensa a errores.
- **Migraciones acopladas al arranque de la app** (`RunMigrations()` corre en cada `init()`): funciona para un proyecto pequeño, pero no versiona cambios de esquema ni permite rollback. Migrar a una herramienta dedicada (`golang-migrate`, `goose`, etc.) especialmente porque ya hay columnas en producción (`padres.celular`, `padres.recibe_whatsapp`) que no están en el código de migración — riesgo de que un entorno nuevo no las tenga.
- **URL del backend hardcodeada en dos archivos del frontend** (`axiosConfig.js` y `ReportePublico.jsx`), con un comentario que sugiere (incorrectamente) que se usa una variable de entorno. Centralizar en `import.meta.env.VITE_API_URL` con `.env`/`.env.production` reales.
- **Enrutamiento interno del panel de personal no usa `react-router`**: las pestañas (kiosco, admin, bitácora, reportes) se manejan con estado local, por lo que no son enlazables por URL, no sobreviven a un refresh y rompen el botón "atrás" del navegador. Migrar a rutas anidadas (`/admin`, `/bitacora`, `/reportes`) mejoraría UX y permitiría proteger rutas con guards declarativos en vez de un modal de PIN imperativo.
- **Configuración de PWA triplicada y potencialmente conflictiva**: manifest inline en `vite.config.js`, `public/manifest.json` estático, y registro manual de `/sw.js` en `index.html` compitiendo con el auto-registro de `vite-plugin-pwa`. Elegir una sola fuente (recomendado: dejar que `vite-plugin-pwa` genere y registre todo, eliminar el `<script>` manual y el manifest estático redundante).
- **Duplicación de UI entre `ReportePublico.jsx` y `VistaPadreDetalle.jsx`**: ambos renderizan prácticamente la misma vista de bitácora diaria (comidas, siesta, esfínter, galería de fotos con modal). Extraer un componente presentacional compartido (`ReporteDiario`) parametrizado por la fuente de datos (pública vs. autenticada).
- **Lógica de fecha "hoy" duplicada de tres formas distintas** (ver bugs arriba) — crear un módulo `utils/fecha.js` con una única función `hoyLocal()` que use siempre `America/Mazatlan` (o la zona horaria configurable de la guardería) y reutilizarla en todos los componentes.
- **Colores/tema hardcodeados como clases de Tailwind literales** en vez de usar `theme.extend.colors` — cambiar el color de marca hoy requeriría editar cientos de ocurrencias de `violet-600` en el código. Definir tokens de color en `tailwind.config.js`.

## 4. Experiencia de usuario y manejo de errores

- **Mezcla inconsistente de `alert()`/`window.confirm()` nativos y `SweetAlert2`**: la librería `sweetalert2` está instalada pero solo se usa en `VistaBitacora.jsx`; el resto de la app (incluyendo confirmaciones destructivas como desactivar/desvincular un niño en `GestionHijos.jsx`) usa diálogos nativos del navegador, con estilos y UX inconsistentes. Unificar en un solo patrón.
- **Estados de carga declarados pero no usados**: `PanelReportes.jsx` mantiene `loading` en el estado pero nunca lo renderiza — el usuario no tiene feedback visual mientras se cargan los reportes.
- **Errores silenciosos**: varios `catch` solo hacen `console.error`/`console.warn` sin mostrar nada al usuario (ej. `DashboardPadre.jsx` si falla la carga de hijos, deja una lista vacía indistinguible de "no tiene hijos"). Diferenciar explícitamente estado vacío vs. error de red.
- **Logs de depuración con emojis dejados en producción** en `FormularioBitacora.jsx` (múltiples `console.log`/`console.warn`) — limpiar antes de cualquier release, y considerar una librería de logging con niveles (silenciable en producción) si se necesita diagnóstico futuro.
- **Accesibilidad**: el viewport deshabilita el zoom (`user-scalable=no, maximum-scale=1.0`), lo cual dificulta el uso a personas con baja visión; y varios botones de solo-ícono carecen de `aria-label`/`title` de forma consistente. Revisar con un lector de pantalla al menos el flujo de login y el kiosco.
- **Falta de validación de fechas en `PanelReportes.jsx`**: no se valida que `fechaInicio <= fechaFin` antes de disparar la consulta, lo que puede producir resultados vacíos confusos sin explicación al usuario.
- **Sin confirmación visual de "no hay rostro en cuadro"** antes de enviar la foto a Rekognition — agregar una validación básica en cliente (o feedback más rico) reduciría llamadas fallidas costosas a AWS.

## 5. Multi-tenancy y modelo de datos

- **Columnas usadas en queries pero ausentes de las migraciones versionadas** (`padres.celular`, `padres.recibe_whatsapp`): documentarlas explícitamente en `RunMigrations()` para que un entorno nuevo (staging, DR) no falle silenciosamente al ejecutar `/seguimiento`.
- **Sin índices en columnas de filtrado multi-tenant frecuentes** (`guarderia_id` en `hijos`, `padres`, `tutor_hijos`, `seguimiento_diario`) más allá del índice existente en `asistencia.fecha_hora`. A medida que crezca el número de guarderías/registros, estas consultas se degradarán. Agregar índices compuestos, p. ej. `(guarderia_id, activo)` en `hijos`.
- **Sin soft-delete/auditoría en `padres` ni en `asistencia`**: solo `hijos` tiene bandera `activo`. Si se requiere trazabilidad (por ejemplo, para disputas sobre quién retiró a un niño), considerar tablas de auditoría o triggers.

## 6. Infraestructura y operación

- **Sin CI/CD**: no hay GitHub Actions ni pipeline equivalente para correr lint/tests/build antes de desplegar. Agregar un workflow mínimo (`go vet`/`go build`, `npm run lint`/`npm run build`) evitaría que errores triviales lleguen a producción.
- **Sin Dockerfile ni infra-as-code versionada**: el despliegue en Render depende de configuración manual no documentada en el repo. Documentar (o versionar) al menos las variables de entorno requeridas (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `DATABASE_URL`, `DATABASE_URL_AUTH`, futuro `JWT_SECRET`) en un `.env.example` y un `README` de despliegue.
- **README del frontend es el genérico de Vite** (no describe el proyecto real). Escribir un README propio con instrucciones de instalación, variables de entorno y arquitectura, y agregar uno equivalente al backend (actualmente no tiene ninguno).
- **Sin monitoreo/alertas** visibles (ej. Sentry, logs estructurados) para detectar fallos de Rekognition/S3/DB en producción de forma proactiva.

## 7. Resumen de prioridades sugerido

1. **Urgente (seguridad)**: mover el JWT secret a variable de entorno, eliminar el log de contraseñas en texto plano, restringir CORS, agregar rate limiting a `/login`, `/identificar` y `/verificar-pin`.
2. **Corto plazo (bugs)**: corregir el typo de formato de hora, unificar el cálculo de "fecha de hoy", arreglar el interceptor 401 para limpiar todo `localStorage`, corregir/limpiar los props no usados.
3. **Mediano plazo (calidad)**: dividir `main.go` en paquetes, introducir pruebas automatizadas, migrar a `react-router` para las pestañas internas, centralizar la URL del backend vía variables de entorno de Vite.
4. **Largo plazo (escalabilidad/DX)**: herramienta de migraciones dedicada, CI/CD, monitoreo, documentación de despliegue, revisión de accesibilidad.
