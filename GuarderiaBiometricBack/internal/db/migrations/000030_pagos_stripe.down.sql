DROP INDEX IF EXISTS pagos_stripe_checkout_session_id_key;
ALTER TABLE pagos DROP COLUMN stripe_payment_intent_id;
ALTER TABLE pagos DROP COLUMN stripe_checkout_session_id;
