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
FROM docker.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/servidor ./servidor

# Las migraciones (internal/db/migrations/*.sql) van embebidas en el binario
# vía go:embed — no hace falta copiar nada más.
EXPOSE 8099
ENTRYPOINT ["./servidor"]
