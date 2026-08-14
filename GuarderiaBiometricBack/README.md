# BioSafe — Backend

API en Go (Gin) para el sistema biométrico de control de asistencia y administración de guarderías. Identificación facial (AWS Rekognition), fotos de bitácora (AWS S3), base de datos PostgreSQL, notificaciones push (Web Push/VAPID, sin servicios de terceros).

## Requisitos

- Go 1.24+
- PostgreSQL 14+ (local o remoto)
- Una cuenta de AWS con acceso a **Rekognition** y **S3** (el servidor no arranca sin credenciales configuradas)

## Configuración

1. Copia `.env.example` a `.env` (o exporta las variables directamente en tu shell/plataforma de despliegue — Go no carga `.env` automáticamente, este archivo es solo referencia).
2. Variables obligatorias para arrancar:
   - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`
   - `JWT_SECRET` — genera una clave larga y aleatoria, por ejemplo con `openssl rand -base64 48`
   - `DATABASE_URL`, `DATABASE_URL_AUTH` — cadenas de conexión a Postgres (pueden apuntar a la misma base)
3. Variables opcionales:
   - `ALLOWED_ORIGINS` — dominios permitidos por CORS, separados por comas. Si no se define, cae a `http://localhost:5173`/`https://localhost:5173` (uso local). **Importante:** el JWT vive en una cookie `SameSite=None; Secure` (ver `internal/middleware/auth.go`), así que este valor debe ser el origen HTTPS *exacto* del frontend en producción — no funciona con `*` ni con un origen HTTP.
   - `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT` — para notificaciones push. Sin ellas, el servidor arranca igual pero no envía notificaciones. Genera el par de claves una sola vez con `npx web-push generate-vapid-keys`.
   - `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`, `FRONTEND_URL` — pagos en línea de colegiatura (Stripe Checkout). Sin `STRIPE_SECRET_KEY` el servidor arranca igual y la función queda deshabilitada por completo (el botón "Pagar en línea" ni aparece en el portal del padre). Para activarla: crea los productos/precio no hace falta, todo se genera al vuelo — solo necesitas la llave secreta y la publicable de tu cuenta de Stripe, `FRONTEND_URL` (el origen HTTPS del frontend, para las URLs de retorno del Checkout), y dar de alta un endpoint de webhook en el dashboard de Stripe apuntando a `POST /webhooks/stripe` suscrito al evento `checkout.session.completed` (de ahí sale `STRIPE_WEBHOOK_SECRET`). `STRIPE_CURRENCY` es opcional, `mxn` por defecto.

## Correr localmente

```bash
go mod download
go build ./...
# con las variables de entorno exportadas:
go run ./cmd/server
```

El servidor queda escuchando en `:8099`. Las migraciones (`internal/db/migrations/*.sql`, versionadas con [golang-migrate](https://github.com/golang-migrate/migrate) y embebidas en el binario) se ejecutan automáticamente al arrancar, registrando en la tabla `schema_migrations` qué versión quedó aplicada. Son seguras de correr sobre una base ya existente (usan `CREATE ... IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS`), incluida una creada antes de este cambio: golang-migrate simplemente registra la versión sin recrear nada.

## Estructura del código

El código está organizado en paquetes Go reales (no solo archivos sueltos en `package main`):

| Ruta | Contenido |
|---|---|
| `cmd/server/main.go` | Punto de entrada: lee configuración, conecta AWS/Postgres, corre migraciones, arranca el router |
| `internal/config` | Lectura de variables de entorno |
| `internal/db` | Migraciones versionadas (`migrations/*.sql`, aplicadas con golang-migrate) |
| `internal/middleware` | `Auth()` (JWT), `RequireStaff()`, CORS, limitador de peticiones en memoria |
| `internal/server` | El `Server` (conexiones inyectadas) y los handlers HTTP, agrupados por dominio: `auth.go` (login, PIN), `asistencia.go` (kiosco biométrico), `hijos.go`, `bitacora.go`, `perfiles.go`, `pagos.go`, `reportes.go`, `push.go` |

Cada handler es un método de `*Server` (ej. `func (s *Server) handleLogin(...)`), con `s.DB`/`s.DBAuth`/`s.Rek`/`s.JWTKey` inyectados una sola vez en `cmd/server/main.go` — no hay variables globales sueltas. `internal/server` tiene pruebas automatizadas (`go test ./...`) que montan el router real con una base de datos simulada (`sqlmock`), sin depender de una Postgres real.

## Modelo de datos (resumen)

`guarderias` (multi-tenant) → `usuarios` (login, roles `admin`/`staff`/`papa`), `padres` (tutores registrados por rostro) ↔ `tutor_hijos` ↔ `hijos`, `asistencia` (entradas/salidas), `seguimiento_diario` + `fotos_seguimiento` (bitácora), `pagos`, `push_subscripciones`.

## Seguridad

- Contraseñas con `bcrypt`, sesiones con JWT (24h).
- PIN de 4 dígitos para desbloquear pestañas administrativas desde el frontend.
- `RequireStaff()` bloquea a nivel de servidor los endpoints que exponen datos de **todas** las familias (perfiles, pagos, reportes) a cualquier cuenta que no sea `admin`/`staff`.
- Rate limiting en memoria (por IP) en los endpoints más sensibles a fuerza bruta o abuso de costos de AWS.
- **Nota**: el rate limiter es en memoria — si el backend se despliega con más de una réplica, hay que moverlo a un almacén compartido (Redis) para que siga siendo efectivo.
- **Bucket de S3 (`biosafe-storage-fotos`) debe ser privado.** El backend nunca sube fotos como públicas y siempre las sirve con URLs firmadas de 1 hora (`internal/server/soporte.go`, `firmarURLFoto`), pero eso no alcanza si el bucket permite lectura pública a nivel de bucket. Configura esto una sola vez en la consola de AWS:
  **S3 → bucket `biosafe-storage-fotos` → Permissions → "Block public access (bucket settings)" → Edit → activar las 4 casillas ("Block all public access") → guardar.**
  Ese ajuste anula cualquier ACL pública en objetos individuales — incluidas fotos subidas antes de este cambio — sin tener que tocarlas una por una.

## Despliegue

Pensado para correr en un solo proceso (ej. Render, Fly.io, un VPS). No requiere Redis, colas ni workers aparte — el cron de cierre nocturno y el envío de notificaciones push corren dentro del mismo proceso.
