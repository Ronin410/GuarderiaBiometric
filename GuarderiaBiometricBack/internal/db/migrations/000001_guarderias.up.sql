CREATE TABLE IF NOT EXISTS guarderias (
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    direccion TEXT,
    plan_suscripcion VARCHAR(20) DEFAULT 'basico',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
