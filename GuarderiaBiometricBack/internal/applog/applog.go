// Package applog centraliza el logging del backend en un formato
// estandarizado -- "quiero logs especializados, estandarizados, para poder
// identificar más fácil si hay algún error" (sin agregar todavía un
// monitoreo de terceros tipo Sentry).
//
// Antes cada handler llamaba log.Printf/log.Println con su propio texto
// libre, mezclado sin distinción entre errores reales, advertencias e
// información -- para saber si una línea del log era un error de verdad
// había que leerla completa y adivinar por las palabras ("Error...", "No se
// pudo...", etc.). Esto usa log/slog (parte de la librería estándar desde
// Go 1.21, sin dependencia nueva) con salida en JSON: cada línea trae
// "level" ("ERROR"/"WARN"/"INFO") como campo aparte, además de "msg" y
// cualquier dato de contexto (guardería, usuario, ruta...). Render (como
// cualquier plataforma tipo PaaS) captura stdout como logs, y su buscador
// de logs permite filtrar por texto -- basta con buscar `"level":"ERROR"`
// para ver solo errores reales, sin importar de qué parte de la app vengan
// ni qué tan distinto sea el mensaje de cada uno.
package applog

import (
	"log/slog"
	"os"
)

var logger = newRespaldo()

// newRespaldo da un logger de texto plano utilizable antes de Init() (por
// ejemplo, en tests que no pasan por cmd/server/main.go) -- mejor eso que
// un nil pointer si algo loguea antes de tiempo.
func newRespaldo() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// Init configura el logger global de la app en JSON. Se llama una sola vez,
// al arrancar (cmd/server/main.go).
func Init() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

// Error registra un error real -- lo que antes era un log.Printf con
// "Error..." o "No se pudo...". args son pares clave-valor de contexto
// (ej. "hijo_id", hijoID) que antes iban embebidos en el texto del mensaje;
// separarlos permite buscar/filtrar por ellos en vez de tener que
// parsear texto libre.
func Error(msg string, err error, args ...any) {
	logger.Error(msg, append([]any{"error", errTexto(err)}, args...)...)
}

// Warn registra una advertencia -- algo que no es un error pero vale la
// pena revisar (ej. una configuración opcional que falta).
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Info registra un evento informativo normal (ej. "login exitoso"), para
// que quede en el mismo formato que el resto en vez de mezclarse como texto
// plano suelto en medio de líneas JSON.
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func errTexto(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
