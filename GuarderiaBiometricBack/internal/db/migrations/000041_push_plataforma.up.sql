-- Suscripción push del DUEÑO de la plataforma (Alejandro), NO de una
-- guardería -- por eso es una tabla aparte de push_subscripciones (que
-- siempre trae padre_id/personal_id + guarderia_id de alguien de una
-- guardería en particular). Aquí no hay ninguno de esos ids: la sesión de
-- /plataforma se autentica con PLATFORM_ADMIN_KEY, no con el JWT normal
-- (ver middleware.RequirePlatformKey), así que no hay "usuario" al que
-- amarrar la suscripción.
CREATE TABLE IF NOT EXISTS push_suscripciones_plataforma (
    id SERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
