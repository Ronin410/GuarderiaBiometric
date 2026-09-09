# Levantar GuarderiaBiometric con Docker

Equivalente en Docker de `podman/run.sh` — mismos Dockerfiles, mismo seed
de datos de prueba, mismos puertos. Si ya tienes Podman instalado, usa
`podman/` en su lugar (`../podman/README.md`); esto es para cuando lo que
tienes es Docker Desktop / Docker Engine.

Requisitos: Docker con Docker Compose v2 (`docker compose`, con espacio —
ya viene incluido en Docker Desktop y en instalaciones recientes de Docker
Engine; si solo tienes `docker-compose` con guion, es la v1 y este archivo
también le sirve, cambia el comando).

## Levantar todo

```bash
cd docker
cp backend.env.example backend.env   # una sola vez — edítalo libremente después
docker compose up --build
```

Esto construye las imágenes del backend y del frontend, levanta Postgres,
espera a que el backend responda (aplica las migraciones solo al
arrancar) y aplica `seed_dev.sql`. La primera vez tarda unos minutos
(descarga imágenes base + `npm ci` + `go mod download`); las siguientes
son mucho más rápidas por el caché de capas de Docker.

Déjalo corriendo en esa terminal, o agrega `-d` para que quede en segundo
plano (`docker compose up --build -d`).

Abre **http://localhost:5173**.

## Usuarios de prueba

Contraseña para los tres: **`Demo1234!`**

| Usuario | Rol | Para qué |
|---|---|---|
| `admin_demo` | admin | Panel completo, PIN `1234` |
| `staff_demo` | staff | Panel completo salvo lo exclusivo de admin, PIN `1234` |
| `papa_demo` | papa | Portal del padre (hijo vinculado: Emiliano Demo) |

## Qué SÍ y qué NO vas a poder probar así

Por defecto las credenciales de AWS son falsas (`dummy`). Con eso:

- **Sí funciona**: login, Familia (buscar/editar tutores, exportar/eliminar
  datos ARCO), Bitácora, Perfiles, Pagos, Reportes, Estadísticas,
  Configuración (Aviso de Privacidad).
- **No funciona**: el kiosco de reconocimiento facial (pestañas Kiosco/
  Registro — llaman a Rekognition de verdad) y ver/subir fotos de la
  bitácora (llaman a S3 de verdad). Fallan con un error controlado, no
  tumban nada.

Para probar con Rekognition/S3 reales, edita `docker/backend.env` con
credenciales de AWS de verdad y el bucket ya creado (ver
`GuarderiaBiometricBack/README.md`), y `docker compose up --build` de
nuevo.

## Para bajar todo

```bash
docker compose down        # conserva los datos de Postgres
docker compose down -v     # además borra el volumen guarderia-pgdata
```

## Notas

- **El seed se vuelve a aplicar cada vez que corres `docker compose up`.**
  Si ya tenías datos de una corrida anterior (el volumen `guarderia-pgdata`
  persiste entre corridas), vas a ver errores de llave duplicada en los
  logs del contenedor de un solo uso `guarderia-seed` — no rompen nada más,
  simplemente ese contenedor termina con error y los demás siguen
  corriendo. Si quieres partir de datos limpios, `docker compose down -v`
  primero.
- **`VITE_API_URL` se hornea en la imagen del frontend en tiempo de
  build** — si cambias el puerto del backend, hay que reconstruir esa
  imagen (`docker compose build frontend`), no basta con reiniciar el
  contenedor.
- Postgres queda expuesto al host en el puerto **5433** (no 5432, a
  propósito, para no chocar con un Postgres local que ya tengas corriendo)
  por conveniencia, para conectarte con un cliente de Postgres local — no
  lo publiques así en un entorno que no sea tu máquina de desarrollo.
- `backend.env` no se sube a git (ver `.gitignore` de la raíz del repo).
- Diferencia de fondo frente al pod de Podman: aquí cada contenedor tiene
  su propia red y se habla con los demás por el **nombre del servicio**
  (`postgres`, `backend`) vía el DNS interno de Compose — no por
  `127.0.0.1` compartido, que es como funciona un pod. Por eso
  `backend.env.example` de esta carpeta trae `DATABASE_URL` distinto al de
  `podman/backend.env.example`.
