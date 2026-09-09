#!/usr/bin/env bash
# Detiene y borra el pod y sus contenedores. El volumen de datos de Postgres
# (guarderia-pgdata) NO se toca — vuelve a estar disponible la próxima vez
# que corras run.sh. Para borrarlo también: podman volume rm guarderia-pgdata
set -uo pipefail

podman pod rm -f guarderia-pod >/dev/null 2>&1
echo "Pod y contenedores eliminados. Los datos de Postgres se conservan."
