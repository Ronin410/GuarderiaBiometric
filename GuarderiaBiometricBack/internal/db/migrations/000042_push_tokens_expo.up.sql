-- Notificaciones push de la app nativa (GuarderiaBiometricMobile, Expo/React
-- Native) -- distintas de push_subscripciones (Web Push/VAPID, para la PWA
-- del navegador). Expo tiene su propio servicio de push que enruta a FCM/
-- APNs por debajo, así que del lado del servidor basta con guardar el
-- "Expo push token" (una cadena, no un endpoint+llaves como Web Push) y
-- mandarle las notificaciones a la API REST de Expo -- ver push_expo.go.
--
-- Mismo criterio que push_subscripciones (000012/000038): padre_id/
-- personal_id son opcionales, cada fila trae UNO de los dos (nunca ambos ni
-- ninguno, decidido en el INSERT según el rol de quien se registra); ninguno
-- lleva FK a usuarios(id) porque esa tabla vive conceptualmente en la base
-- de auth.
CREATE TABLE IF NOT EXISTS push_tokens_expo (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER,
    personal_id INTEGER,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_push_expo_padre ON push_tokens_expo(padre_id);
CREATE INDEX IF NOT EXISTS idx_push_expo_personal ON push_tokens_expo(personal_id);
CREATE INDEX IF NOT EXISTS idx_push_expo_guarderia ON push_tokens_expo(guarderia_id);
