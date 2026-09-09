package middleware

import (
	"strings"

	"biometrico/internal/applog"
)

// ParseAllowedOrigins lee ALLOWED_ORIGINS (lista separada por comas) y, si no está
// configurada, cae a orígenes de desarrollo local para no romper el flujo local con HTTPS.
func ParseAllowedOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		applog.Warn("ALLOWED_ORIGINS no configurada; usando orígenes de desarrollo por defecto (localhost). Configúrala en producción con los dominios reales del frontend")
		return []string{"http://localhost:5173", "https://localhost:5173"}
	}

	partes := strings.Split(raw, ",")
	origenes := make([]string, 0, len(partes))
	for _, p := range partes {
		p = strings.TrimSpace(p)
		if p != "" {
			origenes = append(origenes, p)
		}
	}
	return origenes
}
