// Package server contiene el estado compartido del backend (conexiones a
// Postgres/AWS, clave JWT, configuración de push) y los handlers HTTP,
// agrupados por dominio en archivos separados. Reemplaza las variables
// globales que antes vivían sueltas en package main.
package server

import (
	"database/sql"
	"fmt"
	"time"

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

	loginLimiter       *middleware.RateLimiter
	pinLimiter         *middleware.RateLimiter
	identificarLimiter *middleware.RateLimiter
}

// New crea un Server con sus limitadores de tasa ya inicializados. Las
// conexiones (DB, DBAuth, Rek, S3) y credenciales (JWTKey, Vapid*) se asignan
// después, una vez que cmd/server/main.go termina de leer la configuración.
func New() *Server {
	return &Server{
		loginLimiter:       middleware.NewRateLimiter(10, time.Minute),
		pinLimiter:         middleware.NewRateLimiter(5, time.Minute),
		identificarLimiter: middleware.NewRateLimiter(30, time.Minute),
	}
}

// PushConfigurado indica si el servidor tiene claves VAPID propias. Sin ellas,
// las notificaciones simplemente se omiten (no es un requisito para operar el
// resto de la app, a diferencia de JWT_SECRET).
func (s *Server) PushConfigurado() bool {
	return s.VapidPublicKey != "" && s.VapidPrivateKey != ""
}

// RegisterRoutes registra todas las rutas del backend sobre un *gin.Engine
// nuevo. No arranca el servidor HTTP — eso lo hace cmd/server/main.go — así
// las pruebas pueden montar el router con dependencias falsas (sqlmock) y
// ejercitarlo con httptest, igual que antes hacía setupRouter() en main.go.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	s.registrarRutasAuth(r)
	s.registrarRutasPersonal(r)
	s.registrarRutasAsistencia(r)
	s.registrarRutasHijos(r)
	s.registrarRutasBitacora(r)
	s.registrarRutasPerfiles(r)
	s.registrarRutasGrupos(r)
	s.registrarRutasDocumentos(r)
	s.registrarRutasMenu(r)
	s.registrarRutasCirculares(r)
	s.registrarRutasRecibos(r)
	s.registrarRutasHorarios(r)
	s.registrarRutasChat(r)
	s.registrarRutasAusencias(r)
	s.registrarRutasPagos(r)
	s.registrarRutasReportes(r)
	s.registrarRutasPush(r)
	s.registrarRutasPrivacidad(r)
	s.registrarRutasArco(r)
}
