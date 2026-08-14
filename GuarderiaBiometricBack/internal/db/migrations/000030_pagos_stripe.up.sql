-- Prepara la tabla de pagos para reconciliar pagos hechos en línea vía
-- Stripe Checkout ("Pagos móviles" del PDF) -- sin activar nada todavía:
-- ver server.StripeHabilitado() en pagos_online.go, que se apaga solo si
-- STRIPE_SECRET_KEY no está configurada.
--
-- stripe_checkout_session_id es UNIQUE (permitiendo NULL en cualquier fila
-- de pago manual capturado por staff) para que un webhook reintentado
-- --Stripe reintenta si no responde 200 rápido-- nunca inserte el mismo
-- pago dos veces.
ALTER TABLE pagos ADD COLUMN stripe_checkout_session_id TEXT;
ALTER TABLE pagos ADD COLUMN stripe_payment_intent_id TEXT;
CREATE UNIQUE INDEX pagos_stripe_checkout_session_id_key
    ON pagos (stripe_checkout_session_id)
    WHERE stripe_checkout_session_id IS NOT NULL;
