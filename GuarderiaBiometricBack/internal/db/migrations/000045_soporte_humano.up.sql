-- "Quiero un botón para que quien quiera hablar con un administrador o
-- alguien de soporte no le conteste la IA".
--
-- La bandera vive en la conversación y no en cada mensaje: quien pide hablar
-- con una persona no lo pide para UN mensaje, lo pide para la conversación.
-- Si fuera por mensaje, el siguiente que escribiera le volvería a contestar
-- el asistente y tendría que pedirlo otra vez.
--
-- También se prende sola cuando el dueño de la plataforma contesta en
-- persona: una vez que hay alguien real del otro lado, el asistente se
-- quita de en medio en vez de meterse entre dos personas conversando.
ALTER TABLE conversaciones_soporte
    ADD COLUMN IF NOT EXISTS atendida_por_humano BOOLEAN NOT NULL DEFAULT false;
