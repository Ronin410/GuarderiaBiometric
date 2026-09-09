# Build multi-stage: binario Go estático, sin cgo (todas las dependencias de
# este proyecto son Go puro — lib/pq, gin, aws-sdk-go-v2, bcrypt — así que
# CGO_ENABLED=0 produce un binario que corre en una imagen final mínima).
FROM docker.io/library/golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/servidor ./cmd/server

# Imagen final: alpine en vez de scratch/distroless porque el backend
# necesita:
#   - ca-certificates: llamadas HTTPS a AWS (Rekognition/S3) y a Postgres
#     administrado con TLS.
#   - tzdata: internal/server/fecha.go usa time.LoadLocation("America/Mazatlan")
#     para calcular "hoy" en la zona horaria de la guardería — sin el paquete
#     de zonas horarias del sistema, esa llamada falla.
#   - openssl: solo para generar el certificado autofirmado de abajo, no lo
#     usa el binario en tiempo de ejecución.
FROM docker.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates tzdata openssl
WORKDIR /app
COPY --from=build /out/servidor ./servidor

# Certificado autofirmado horneado en la imagen (no montado desde el host):
# la cookie de sesión es Secure (ver internal/middleware/auth.go), así que
# sin HTTPS el navegador la descarta. En Render eso lo resuelve la
# plataforma; en un contenedor local no hay nada delante, así que el propio
# binario sirve TLS con esto. Va horneado y no montado por volumen a
# propósito: en Podman Desktop sobre Windows con máquina Hyper-V, compartir
# una carpeta del host con la VM es un paso aparte que suele fallar — un
# certificado nuevo por imagen evita depender de eso. Es de un solo uso
# (desarrollo local), no pasa nada si cambia en cada build.
RUN mkdir -p /certs && \
    openssl req -x509 -newkey rsa:2048 -keyout /certs/key.pem -out /certs/cert.pem \
      -days 825 -nodes -subj "/CN=localhost"
ENV TLS_CERT_FILE=/certs/cert.pem
ENV TLS_KEY_FILE=/certs/key.pem

# Las migraciones (internal/db/migrations/*.sql) van embebidas en el binario
# vía go:embed — no hace falta copiar nada más.
EXPOSE 8099
ENTRYPOINT ["./servidor"]
