#!/usr/bin/env bash
# Levanta Postgres + backend + frontend con Podman, sin podman-compose: un
# pod (los contenedores comparten "localhost" entre sí, igual que haría una
# red de docker-compose) más podman run/build directos.
#
# Uso:
#   ./run.sh          construye las imágenes y levanta todo desde cero
#   ./stop.sh         detiene y borra los contenedores (conserva los datos)
set -euo pipefail
cd "$(dirname "$0")"

# Git Bash en Windows (MSYS) convierte solo argumentos que empiezan con "/"
# a rutas de Windows sin que se lo pidas — rompe los "-v /host:/contenedor"
# de más abajo. En Linux/macOS/WSL esta variable simplemente no se usa para
# nada, así que es seguro dejarla siempre.
export MSYS_NO_PATHCONV=1

POD_NAME=guarderia-pod
IMG_BACKEND=guarderia-backend:local
IMG_FRONTEND=guarderia-frontend:local
VOL_PG=guarderia-pgdata

for cmd in podman curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Falta '$cmd' — instálalo antes de continuar." >&2; exit 1; }
done

# 1. Variables de entorno del backend ------------------------------------------
if [ ! -f backend.env ]; then
  echo "==> No existe podman/backend.env, copiando desde backend.env.example."
  cp backend.env.example backend.env
fi

# 2. Limpieza de una corrida anterior -------------------------------------------
./stop.sh >/dev/null 2>&1 || true

# 3. Pod compartido ------------------------------------------------------------
# Solo estos tres puertos quedan expuestos al host; entre ellos los
# contenedores se hablan por localhost (comparten la red del pod). Postgres
# va en 5433, no 5432, para no chocar con un Postgres local que ya tengas
# corriendo en tu máquina — el backend igual se conecta por dentro del pod
# a 127.0.0.1:5432 (ver backend.env), eso no pasa por este puerto expuesto.
echo "==> Creando pod..."
podman pod create --name "$POD_NAME" \
  -p 5173:8080 \
  -p 8099:8099 \
  -p 5433:5432

# 4. Postgres -------------------------------------------------------------------
echo "==> Levantando Postgres..."
podman run -d --pod "$POD_NAME" --name guarderia-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=guarderia \
  -v "${VOL_PG}:/var/lib/postgresql/data" \
  docker.io/library/postgres:16-alpine

echo -n "==> Esperando a que Postgres acepte conexiones"
until podman exec guarderia-postgres pg_isready -U postgres >/dev/null 2>&1; do
  echo -n "."
  sleep 1
done
echo " listo."

# 5. Backend ----------------------------------------------------------------------
echo "==> Construyendo imagen del backend..."
podman build -t "$IMG_BACKEND" -f ../GuarderiaBiometricBack/Dockerfile ../GuarderiaBiometricBack

echo "==> Levantando backend..."
# El certificado TLS ya viene horneado en la imagen (ver el Dockerfile) —
# no hace falta montar nada del host.
podman run -d --pod "$POD_NAME" --name guarderia-backend \
  --env-file backend.env \
  "$IMG_BACKEND"

echo -n "==> Esperando a que el backend responda (aplica migraciones al arrancar)"
tries=0
until curl -sk -o /dev/null https://localhost:8099/aviso-privacidad; do
  echo -n "."
  sleep 1
  tries=$((tries + 1))
  if [ "$tries" -gt 60 ]; then
    echo
    echo "El backend no respondió a tiempo. Revisa los logs: podman logs guarderia-backend" >&2
    exit 1
  fi
done
echo " listo."

# 6. Datos de prueba ---------------------------------------------------------------
echo "==> Aplicando datos de prueba..."
podman exec -i guarderia-postgres psql -U postgres -d guarderia \
  < ../GuarderiaBiometricBack/internal/db/seeds/seed_dev.sql

# 7. Frontend -----------------------------------------------------------------------
echo "==> Construyendo imagen del frontend..."
podman build -t "$IMG_FRONTEND" \
  --build-arg VITE_API_URL=https://localhost:8099 \
  -f ../GuarderiaBiometricFront/Dockerfile ../GuarderiaBiometricFront

echo "==> Levantando frontend..."
podman run -d --pod "$POD_NAME" --name guarderia-frontend "$IMG_FRONTEND"

cat <<'EOF'

======================================================================
Listo. Abre: http://localhost:5173

Usuarios de prueba (contraseña para los tres: Demo1234!):
  admin_demo   (rol admin, PIN 1234)
  staff_demo   (rol staff, PIN 1234)
  papa_demo    (rol papa, portal del padre — hijo vinculado: Emiliano Demo)

El kiosco de reconocimiento facial (pestañas Kiosco/Registro) NO va a
funcionar con estos datos: los tutores sembrados tienen un face_id de
mentira, nunca pasaron por Rekognition. Todo lo demás (Familia, Bitácora,
Perfiles, Pagos, Reportes, Estadísticas, Configuración, ARCO) sí.

Backend directo, para curl/depuración (certificado autofirmado, usa -k):
  https://localhost:8099

Para detener todo (conserva los datos):  ./stop.sh
Para borrar también los datos:           podman volume rm guarderia-pgdata
======================================================================
EOF
