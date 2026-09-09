# Build multi-stage: la SPA se compila una sola vez y queda servida por
# nginx — no hace falta Node en la imagen final.
FROM docker.io/library/node:20-alpine AS build

WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# VITE_API_URL se hornea en el bundle en tiempo de build (Vite reemplaza
# import.meta.env.* de forma estática) — no es una variable que se pueda
# cambiar al arrancar el contenedor, hay que reconstruir la imagen si cambia.
ARG VITE_API_URL
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM docker.io/library/nginx:1.27-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080
