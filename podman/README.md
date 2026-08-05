# Levantar GuarderiaBiometric con Podman (sin podman-compose)

No usa `podman-compose`. Hay dos formas, usa la que te acomode:

- **Opción A — script (`run.sh`)**: Linux, macOS, o Windows con WSL.
- **Opción B — GUI de Podman Desktop (`kube.yaml`)**: Windows sin WSL
  (máquina Hyper-V) u otro caso donde prefieras clics en vez de terminal.

Ambas usan las mismas imágenes/Dockerfiles y el mismo seed — solo cambia
cómo las levantas.

---

## Opción A: script

Requisitos: `podman`, `curl`.

```bash
cd podman
./run.sh
```

Esto: copia `backend.env.example` a `backend.env` si no existe (una sola
vez — edítalo libremente después, no se sobreescribe), crea el pod, levanta
Postgres, construye y levanta el backend (que aplica las migraciones al
arrancar), aplica los datos de prueba, y construye y levanta el frontend.

```bash
./stop.sh   # detiene y borra los contenedores; conserva los datos
```

---

## Opción B: GUI de Podman Desktop

Para cuando no puedes usar una terminal bash (ej. Windows sin WSL, con
Podman Desktop sobre una máquina Hyper-V).

**1. Construir la imagen del backend**
Pestaña **Images** → **Build an image** (o el botón `+`):
- Containerfile/Dockerfile: `GuarderiaBiometricBack/Dockerfile`
- Build context: la carpeta `GuarderiaBiometricBack`
- Nombre de la imagen: `guarderia-backend:local`
- Build

**2. Construir la imagen del frontend**
Igual, pero:
- Containerfile/Dockerfile: `GuarderiaBiometricFront/Dockerfile`
- Build context: la carpeta `GuarderiaBiometricFront`
- Nombre de la imagen: `guarderia-frontend:local`
- En **Build arguments** (o "Advanced options"), agrega:
  `VITE_API_URL` = `https://localhost:8099`
  (si no ves un campo para esto en tu versión de Podman Desktop, dímelo y
  lo resolvemos por otro lado)
- Build

**3. Levantar todo con el YAML**
Pestaña **Pods** (o el menú `...` de Containers, según tu versión) →
**Play Kubernetes YAML** → selecciona `podman/kube.yaml`.

Esto crea el pod `guarderia-pod` con Postgres + backend + frontend juntos,
con los puertos ya publicados (5173, 8099, 5432). El contenedor del backend
puede reiniciarse solo una o dos veces al principio (arrancó antes de que
Postgres estuviera listo) — en unos segundos se estabiliza solo.

**4. Aplicar los datos de prueba**
Este paso sí necesita una línea de comando — desde Git Bash:
```bash
cd podman
podman exec -i guarderia-postgres psql -U postgres -d guarderia \
  < ../GuarderiaBiometricBack/internal/db/seeds/seed_dev.sql
```
(o desde Podman Desktop: click derecho en el contenedor `guarderia-postgres`
→ abrir una terminal dentro de él, y ahí corres `psql -U postgres -d
guarderia` y pegas el contenido de `seed_dev.sql`).

**Para bajar todo**: pestaña Pods → selecciona `guarderia-pod` → Delete (o
`podman pod rm -f guarderia-pod` desde Git Bash).

---

## Al terminar (cualquiera de las dos opciones)

Abre **http://localhost:5173**.

## Usuarios de prueba

Contraseña para los tres: **`Demo1234!`**

| Usuario | Rol | Para qué |
|---|---|---|
| `admin_demo` | admin | Panel completo, PIN `1234` |
| `staff_demo` | staff | Panel completo salvo lo exclusivo de admin, PIN `1234` |
| `papa_demo` | papa | Portal del padre (hijo vinculado: Emiliano Demo) |

Ya vienen sembrados 3 tutores, 4 niños, asistencia de hoy y ayer, bitácora de
hoy, pagos (uno pagado, uno parcial, uno pendiente, uno vencido) y el Aviso
de Privacidad configurado (con un texto de ejemplo — **no es un aviso legal
real**, reemplázalo desde el panel Configuración antes de usar esto con
datos de verdad).

## Qué SÍ y qué NO vas a poder probar así

Por defecto las credenciales de AWS son falsas (`dummy`). Con eso:

- **Sí funciona**: login, Familia (buscar/editar tutores, exportar/eliminar
  datos ARCO), Bitácora, Perfiles, Pagos, Reportes, Estadísticas,
  Configuración (Aviso de Privacidad).
- **No funciona**: el kiosco de reconocimiento facial (pestañas Kiosco/
  Registro — llaman a Rekognition de verdad) y ver/subir fotos de la
  bitácora (llaman a S3 de verdad). Fallan con un error controlado, no
  tumban nada.

## Probar con Rekognition real

1. Consigue credenciales de una cuenta de AWS con permisos de Rekognition y
   S3, en una región donde ambos servicios estén disponibles.
2. Crea el bucket `biosafe-storage-fotos` en esa región (el nombre está fijo
   en el código, `internal/server/soporte.go`) y actívale **"Block all
   public access"** (ver `GuarderiaBiometricBack/README.md`, sección de
   AWS) — las fotos se sirven por URL firmada, nunca deben ser públicas.
3. **Opción A (script)**: edita `podman/backend.env` — reemplaza
   `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` y `AWS_REGION` con los
   valores reales, luego `./stop.sh && ./run.sh`.
   **Opción B (GUI)**: edita esos mismos tres valores directo en
   `podman/kube.yaml` (contenedor `guarderia-backend`), borra el pod desde
   Podman Desktop y vuelve a jugar el YAML.

Los tutores sembrados por `seed_dev.sql` tienen un `face_id` inventado (nunca
pasaron por Rekognition de verdad), así que el kiosco no los va a reconocer
— para probar el flujo completo, registra un tutor nuevo desde la pestaña
Registro del kiosco.

## Notas

- **El certificado TLS del backend viene horneado en la imagen**
  (`GuarderiaBiometricBack/Dockerfile` lo genera con `openssl` durante el
  build) — no se monta desde el host. Es a propósito: en Podman Desktop
  sobre Windows con máquina Hyper-V, compartir una carpeta del host con la
  VM es un paso aparte que suele fallar, y así te lo evitas.
- **VITE_API_URL se hornea en la imagen del frontend en tiempo de build** —
  si cambias el puerto del backend, hay que reconstruir la imagen del
  frontend, no basta con reiniciar el contenedor.
- El puerto 5432 del pod queda expuesto al host por conveniencia (para
  conectarte con un cliente de Postgres local) — no lo publiques así en un
  entorno que no sea tu máquina de desarrollo.
- `backend.env` no se sube a git (ver `.gitignore` de la raíz del repo).
