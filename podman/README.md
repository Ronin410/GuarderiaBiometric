# Levantar GuarderiaBiometric con Podman (sin podman-compose)

No usa `podman-compose` — es un [pod](https://docs.podman.io/en/latest/markdown/podman-pod.1.html)
de Podman (todos los contenedores comparten `localhost` entre sí, igual que
haría la red que crea `docker-compose`) más `podman build`/`run` directos,
orquestados por `run.sh`.

## Requisitos

- `podman`
- `openssl` (para el certificado autofirmado)
- `curl` (para saber cuándo el backend ya está listo)

## Uso

```bash
cd podman
./run.sh
```

Esto:
1. Genera un certificado autofirmado en `podman/certs/` (una sola vez).
2. Copia `backend.env.example` a `backend.env` si no existe (una sola vez —
   edítalo libremente después, no se sobreescribe).
3. Crea el pod, levanta Postgres, construye y levanta el backend (que aplica
   las migraciones solo al arrancar), aplica los datos de prueba
   (`../GuarderiaBiometricBack/internal/db/seeds/seed_dev.sql`), y construye
   y levanta el frontend.

Al terminar: **http://localhost:5173**

```bash
./stop.sh   # detiene y borra los contenedores; conserva los datos
```

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

Por defecto `backend.env` trae credenciales de AWS falsas (`dummy`). Con eso:

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
3. Edita `podman/backend.env`: reemplaza `AWS_ACCESS_KEY_ID`,
   `AWS_SECRET_ACCESS_KEY` y `AWS_REGION` con los valores reales.
4. `./stop.sh && ./run.sh` para que el backend arranque con las credenciales
   nuevas.

Los tutores sembrados por `seed_dev.sql` tienen un `face_id` inventado (nunca
pasaron por Rekognition de verdad), así que el kiosco no los va a reconocer
— para probar el flujo completo, registra un tutor nuevo desde la pestaña
Registro del kiosco.

## Notas

- **VITE_API_URL se hornea en la imagen del frontend en tiempo de build**
  (`run.sh` la pasa como `--build-arg`) — si cambias el puerto del backend,
  hay que reconstruir la imagen del frontend, no basta con reiniciar el
  contenedor.
- El puerto 5432 del pod queda expuesto al host por conveniencia (para
  conectarte con un cliente de Postgres local) — no lo publiques así en un
  entorno que no sea tu máquina de desarrollo.
- `backend.env` y `certs/` no se suben a git (ver `.gitignore` de la raíz
  del repo).
