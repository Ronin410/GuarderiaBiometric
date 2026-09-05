CREATE TABLE IF NOT EXISTS usuarios (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    pin_admin VARCHAR(4) NOT NULL,
    rol VARCHAR(20) DEFAULT 'staff' CHECK (rol IN ('admin', 'staff', 'papa')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
