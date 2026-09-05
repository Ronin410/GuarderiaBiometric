package server

// registrarAcceso guarda en logs_acceso quién (si se conoce) hizo qué sobre
// datos sensibles y cuándo — soporte técnico para el hallazgo de auditoría
// de "no hay bitácora de accesos" (LFPDPPP). Es "fire and forget": si falla
// el guardado, se registra en el log de la aplicación pero no se aborta la
// respuesta al usuario — un problema de auditoría no debe tumbar el login,
// la exportación ARCO, etc. (mismo criterio que notificarEvento en push.go).
//
// guarderiaID y usuarioID van como any para poder pasar nil cuando no se
// conocen (ej. un login con un usuario que no existe nunca resuelve
// guarderia_id).
func (s *Server) registrarAcceso(evento string, guarderiaID, usuarioID any, detalle, ip string) {
	if s.DB == nil {
		// Pruebas unitarias que solo montan DBAuth (ej. login_test.go) no
		// tienen por qué wirear también DB solo para poder auditar: se omite
		// el registro en vez de fallar, mismo criterio de "no bloquear el
		// flujo real" que un error de INSERT más abajo.
		return
	}
	_, err := s.DB.Exec(
		`INSERT INTO logs_acceso (evento, guarderia_id, usuario_id, detalle, ip) VALUES ($1, $2, $3, $4, $5)`,
		evento, guarderiaID, usuarioID, detalle, ip,
	)
	if err != nil {
		s.logError(nil, "registrarAcceso: no se pudo guardar el evento de auditoría", err, "evento", evento)
	}
}
