-- "Quiero una forma de bloquear el acceso a una guardería, ya que si no
-- pagan manualmente quiero quitarles el acceso dándoles tiempo para que
-- paguen, pero eso lo haré manualmente" -- un interruptor que el dueño de
-- la plataforma prende/apaga a mano desde /plataforma (ver
-- handleBloquearGuarderia/handleDesbloquearGuarderia en plataforma.go), NO
-- un corte automático por fecha de vencimiento: no hay ningún cron que lo
-- toque.
ALTER TABLE guarderias ADD COLUMN IF NOT EXISTS bloqueada BOOLEAN NOT NULL DEFAULT false;
-- bloqueada_en queda NULL mientras no está bloqueada -- sirve para que
-- /plataforma pueda mostrar desde cuándo lleva bloqueada una guardería.
ALTER TABLE guarderias ADD COLUMN IF NOT EXISTS bloqueada_en TIMESTAMP;
