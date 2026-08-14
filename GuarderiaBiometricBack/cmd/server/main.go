package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"time"

	appdb "biometrico/internal/db"
	"biometrico/internal/middleware"
	"biometrico/internal/server"

	appconfig "biometrico/internal/config"
)

func main() {
	cfg := appconfig.Load()
	srv := conectarServicios(cfg)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     middleware.ParseAllowedOrigins(cfg.AllowedOriginsRaw),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Guarderia-Slug", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	srv.IniciarTareasProgramadas()
	if !srv.PushConfigurado() {
		log.Println("ADVERTENCIA: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY no configuradas; las notificaciones push quedan deshabilitadas.")
	}
	if !srv.StripeHabilitado() {
		log.Println("INFO: STRIPE_SECRET_KEY no configurada; los pagos en línea quedan deshabilitados (el resto de la app no se ve afectado).")
	}

	srv.RegisterRoutes(r)

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Println("Sirviendo HTTPS directamente (TLS_CERT_FILE/TLS_KEY_FILE configurados)")
		log.Fatal(r.RunTLS(":8099", cfg.TLSCertFile, cfg.TLSKeyFile))
	}
	r.Run(":8099")
}

// conectarServicios valida la configuración y abre las conexiones externas
// (AWS, Postgres) antes de levantar el router. Vive aquí, en cmd/server, y no
// en un func init() del paquete server: un init() que hace I/O y puede matar
// el proceso también correría al ejecutar "go test" sobre ese paquete.
func conectarServicios(cfg appconfig.Config) *server.Server {
	if cfg.AWSAccessKeyID == "" || cfg.AWSSecretAccessKey == "" {
		log.Fatal("ERROR: Credenciales de AWS no configuradas.")
	}

	if cfg.JWTSecret == "" {
		log.Fatal("ERROR: JWT_SECRET no configurada. Define esta variable de entorno con una clave secreta segura antes de iniciar el servidor.")
	}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("No se pudo cargar la configuración de AWS: %v", err)
	}

	dbAuth, err := sql.Open("postgres", cfg.DatabaseURLAuth)
	if err != nil || dbAuth.Ping() != nil {
		log.Fatal("Error conectando a DB Auth")
	}

	conexion, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := appdb.RunMigrations(conexion); err != nil {
		log.Fatal(err)
	}

	if err = conexion.Ping(); err != nil {
		log.Fatal("No se pudo conectar a la DB:", err)
	}
	fmt.Println("Conexión a Postgres exitosa")

	srv := server.New()
	srv.DB = conexion
	srv.DBAuth = dbAuth
	srv.Rek = rekognition.NewFromConfig(awsCfg)
	srv.S3 = s3.NewFromConfig(awsCfg)
	srv.JWTKey = []byte(cfg.JWTSecret)
	srv.VapidPublicKey = cfg.VapidPublicKey
	srv.VapidPrivateKey = cfg.VapidPrivateKey
	srv.VapidSubject = cfg.VapidSubject
	srv.StripeSecretKey = cfg.StripeSecretKey
	srv.StripePublishableKey = cfg.StripePublishableKey
	srv.StripeWebhookSecret = cfg.StripeWebhookSecret
	srv.StripeCurrency = cfg.StripeCurrency
	srv.FrontendURL = cfg.FrontendURL
	return srv
}
