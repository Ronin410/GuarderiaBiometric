// Package server contiene el estado compartido del backend (conexiones a
// Postgres/AWS, clave JWT, configuración de push) y los handlers HTTP,
// agrupados por dominio en archivos separados. Reemplaza las variables
// globales que antes vivían sueltas en package main.
package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// getCollectionID calcula el nombre de la colección de Rekognition de una
// guardería. El frontend nunca debe enviar este valor: siempre se recalcula
// aquí a partir del guarderia_id del JWT.
func getCollectionID(guarderiaID any) string {
	return fmt.Sprintf("guarderia-%v", guarderiaID)
}

// Server agrupa todas las dependencias que antes eran variables globales de
// package main (db, dbAuth, rekClient, jwtKey) y las inyecta explícitamente
// en cada handler vía sus métodos.
type Server struct {
	DB     *sql.DB
	DBAuth *sql.DB
	Rek    *rekognition.Client
	S3     *s3.Client
	JWTKey []byte

	VapidPublicKey  string
	VapidPrivateKey string
	VapidSubject    string

	// Pagos móviles (Stripe Checkout) -- ver StripeHabilitado().
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	StripeCurrency       string
	FrontendURL          string

	// Ver middleware.RequirePlatformKey.
	PlatformAdminKey string

	// Chat de soporte con IA (RAG) -- ver RAGSoporteHabilitado() e
	// internal/server/ia_soporte.go. AnthropicClient solo queda construido
	// de verdad cuando AnthropicAPIKey no está vacía (ver cmd/server/main.go);
	// con las claves vacías, RAGSoporteHabilitado() es false y nunca se usa.
	AnthropicAPIKey string
	VoyageAPIKey    string
	AnthropicClient anthropic.Client

	loginLimiter       *middleware.RateLimiter
	pinLimiter         *middleware.RateLimiter
	identificarLimiter *middleware.RateLimiter
	soporteLimiter     *middleware.RateLimiter
}

// New crea un Server con sus limitadores de tasa ya inicializados. Las
// conexiones (DB, DBAuth, Rek, S3) y credenciales (JWTKey, Vapid*) se asignan
// después, una vez que cmd/server/main.go termina de leer la configuración.
func New() *Server {
	return &Server{
		loginLimiter:       middleware.NewRateLimiter(10, time.Minute),
		pinLimiter:         middleware.NewRateLimiter(5, time.Minute),
		identificarLimiter: middleware.NewRateLimiter(30, time.Minute),
		// Chat de soporte: incluye el formulario público de prospectos (sin
		// cuenta), así que necesita su propio límite -- generoso para una
		// conversación real (varios mensajes seguidos), pero suficiente para
		// frenar un bot que intente usarlo para mandar spam.
		soporteLimiter: middleware.NewRateLimiter(20, 5*time.Minute),
	}
}

// PushConfigurado indica si el servidor tiene claves VAPID propias. Sin ellas,
// las notificaciones simplemente se omiten (no es un requisito para operar el
// resto de la app, a diferencia de JWT_SECRET).
func (s *Server) PushConfigurado() bool {
	return s.VapidPublicKey != "" && s.VapidPrivateKey != ""
}

// StripeHabilitado indica si el servidor tiene su llave secreta de Stripe
// configurada. Sin ella, /pagos-online/config le dice al frontend que la
// función no está disponible (oculta el botón de "Pagar en línea" por
// completo) y /padre/pagos-online/checkout responde 501 -- pensado a
// propósito para desplegarse "apagado": el código y la migración quedan
// listos, pero nadie puede cobrar nada hasta que se configuren las
// variables de entorno de Stripe.
func (s *Server) StripeHabilitado() bool {
	return s.StripeSecretKey != ""
}

// RAGSoporteHabilitado indica si el chat de soporte puede intentar
// contestar solo con IA antes de avisarle al dueño de la plataforma (ver
// ia_soporte.go). Hacen falta las dos claves: Voyage para buscar contexto
// (embeddings) y Anthropic para redactar la respuesta -- sin cualquiera de
// las dos, handleEnviarMensajeSoporte se comporta exactamente como antes de
// esta función.
func (s *Server) RAGSoporteHabilitado() bool {
	return s.AnthropicAPIKey != "" && s.VoyageAPIKey != ""
}

// RegisterRoutes registra todas las rutas del backend sobre un *gin.Engine
// nuevo. No arranca el servidor HTTP — eso lo hace cmd/server/main.go — así
// las pruebas pueden montar el router con dependencias falsas (sqlmock) y
// ejercitarlo con httptest, igual que antes hacía setupRouter() en main.go.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	// /health: sin auth, sin tocar la base de datos -- lo que Render (u
	// otra plataforma) necesita para saber "el proceso sigue vivo y
	// responde". No usa /aviso-privacidad ni otra ruta real de la app a
	// propósito: esas exigen sesión o devuelven 401/404 sin ella, lo que un
	// health check interpretaría como "no está sano" aunque sí lo esté.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	s.registrarRutasAuth(r)
	s.registrarRutasPersonal(r)
	s.registrarRutasAsistencia(r)
	s.registrarRutasHijos(r)
	s.registrarRutasBitacora(r)
	s.registrarRutasPerfiles(r)
	s.registrarRutasGrupos(r)
	s.registrarRutasTiposDocumento(r)
	s.registrarRutasDocumentos(r)
	s.registrarRutasMenu(r)
	s.registrarRutasCirculares(r)
	s.registrarRutasRecibos(r)
	s.registrarRutasHorarios(r)
	s.registrarRutasChat(r)
	s.registrarRutasAusencias(r)
	s.registrarRutasCalendario(r)
	s.registrarRutasComedor(r)
	s.registrarRutasEncuestas(r)
	s.registrarRutasGaleria(r)
	s.registrarRutasPagos(r)
	s.registrarRutasPagosOnline(r)
	s.registrarRutasReportes(r)
	s.registrarRutasPush(r)
	s.registrarRutasPushExpo(r)
	s.registrarRutasPrivacidad(r)
	s.registrarRutasGuarderia(r)
	s.registrarRutasArco(r)
	s.registrarRutasSolicitudes(r)
	s.registrarRutasPlataformaGuarderias(r)
	s.registrarRutasSoporte(r)
	s.registrarRutasPushPlataforma(r)
}
