# Desplegar Pasitos en Render (plan gratis)

Para probar la app desde otros dispositivos (celular, tablet, otra
computadora) sin depender de tu máquina local. Usa el plan gratuito de
Render en sus 3 piezas: base de datos, backend y frontend.

## Antes de empezar

Necesitas tener a la mano:
- **Credenciales de AWS** (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
  `AWS_REGION`) con acceso a Rekognition y S3 -- las mismas que ya usas en
  local (`podman/backend.env` o `docker/backend.env`), o crea unas nuevas.
  El backend **no arranca sin ellas** (`log.Fatal` si faltan) -- esto no es
  opcional, a diferencia de Vapid/Stripe.
- Este repo tiene que estar en GitHub (ya lo está) y Render tiene que poder
  leerlo -- lo conectas en el paso siguiente.

## 1. Crear los 3 servicios con el Blueprint

Este repo ya trae `render.yaml` en la raíz, con la base de datos, el
backend y el frontend definidos.

1. Entra a [dashboard.render.com](https://dashboard.render.com) → **New +**
   → **Blueprint**.
2. Conecta tu cuenta de GitHub (si no lo has hecho) y selecciona este repo.
3. Render lee `render.yaml` solo y te muestra los 3 recursos que va a
   crear: `pasitos-db` (Postgres, gratis), `pasitos-backend` (Docker,
   gratis), `pasitos-frontend` (Static Site, gratis). Dale **Apply**.
4. Tarda unos minutos -- el backend en particular, porque compila el
   binario de Go dentro de Docker.

## 2. Rellenar las variables secretas

`render.yaml` deja algunas variables en blanco a propósito (nunca se suben
credenciales reales a git). Una vez creado `pasitos-backend`:

1. Ábrelo en el dashboard → pestaña **Environment**.
2. Rellena:
   - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` --
     **obligatorias**, sin esto el backend no arranca.
   - `PLATFORM_ADMIN_KEY` -- opcional pero recomendada: una clave que tú
     inventes (ej. genera una con `openssl rand -hex 32`) para poder entrar
     a `/plataforma` y aprobar la primera guardería (ver paso 4). Sin ella,
     esa pantalla queda deshabilitada.
   - `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` -- opcional,
     solo si quieres probar notificaciones push (genera el par con
     `npx web-push generate-vapid-keys`).
   - `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` -- opcional, solo
     si quieres el aviso por correo de mensajes nuevos en el chat de
     soporte (ver el botón "Soporte" flotante y la pestaña "Soporte" en
     `/plataforma`). Con Gmail: `smtp.gmail.com`, `587`, tu correo, y una
     **contraseña de aplicación** (myaccount.google.com/apppasswords -- NO
     la contraseña normal de la cuenta, esa Gmail la rechaza para SMTP).
     `PLATFORM_NOTIFY_EMAIL` es opcional -- si lo dejas vacío, el aviso
     llega al mismo `SMTP_USER`.
3. Guarda -- Render vuelve a desplegar el servicio solo.

## 3. Confirmar las URLs reales

`render.yaml` asume que los servicios se van a llamar exactamente
`pasitos-backend` y `pasitos-frontend` (a partir de ahí Render arma
`https://pasitos-backend.onrender.com`, etc.). Si alguno de los dos ya
estaba tomado, Render le agregó un sufijo al nombre.

Revisa la URL real de cada servicio (arriba, en su página del dashboard) y,
si cambió:
- En `pasitos-backend` → Environment: actualiza `ALLOWED_ORIGINS` y
  `FRONTEND_URL` con la URL real del frontend.
- En `pasitos-frontend` → Environment: actualiza `VITE_API_URL` con la URL
  real del backend, y **vuelve a desplegar el frontend** (Manual Deploy →
  Deploy latest commit) -- a diferencia del backend, esta variable se
  hornea en el build, cambiarla sola no alcanza.

## 4. Primera guardería de prueba

No hay datos de ejemplo precargados (a diferencia de `podman/run.sh`, que
sí siembra usuarios de prueba) -- en Render das de alta tu primera
guardería real, usando lo que ya construimos:

1. Abre `https://pasitos-frontend.onrender.com/registro-guarderia` y llena
   el formulario (nombre de la guardería, tu usuario/contraseña de admin).
2. Abre `https://pasitos-frontend.onrender.com/plataforma`, entra con el
   `PLATFORM_ADMIN_KEY` que configuraste en el paso 2, y aprueba la
   solicitud.
3. Ya puedes iniciar sesión normal en `pasitos-frontend.onrender.com` con
   el usuario que registraste.

## 5. Probar desde otro dispositivo

Abre `https://pasitos-frontend.onrender.com` desde tu celular, tablet, o
cualquier otra computadora -- no necesitas estar en la misma red, Render
ya lo sirve público en internet.

## Cosas del plan gratis que vas a notar

- **El backend se duerme tras 15 min sin tráfico**, y tarda ~1 minuto en
  despertar con la siguiente petición -- normal del plan gratis, no es un
  error. El frontend (Static Site) no tiene este problema, siempre responde
  rápido.
- **La base de datos gratis expira 30 días después de creada** (con 14
  días de gracia para actualizarla antes de que se borre). Perfecto para
  probar; si esto se vuelve producción real, hay que pasarla a un plan de
  pago antes de que expire.
- El kiosco de reconocimiento facial (pestañas Recepción/Registro) sí va a
  funcionar en Render, siempre que las credenciales de AWS sean reales y
  tengan permisos de Rekognition/S3 -- a diferencia de correrlo local con
  credenciales `dummy`.

Fuentes sobre los límites del plan gratis: [Render Docs — Free tier](https://render.com/docs/free) · [Render Docs — Web Services](https://render.com/docs/web-services)
