package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"time"

	"biometrico/internal/applog"
	appdb "biometrico/internal/db"
	"biometrico/internal/middleware"
	"biometrico/internal/server"

	appconfig "biometrico/internal/config"
)

func main() {
	applog.Init()

	cfg := appconfig.Load()
	srv := conectarServicios(cfg)

	// gin.New() en vez de gin.Default(): Default() trae su propio logger de
	// texto plano (gin.Logger()), que reemplazamos por
	// middleware.StructuredLogger() -- mismo formato JSON que el resto del
	// logging de la app, para que un 5xx se pueda encontrar buscando
	// "level":"ERROR" sin importar qué ruta lo produjo. gin.Recovery() se
	// mantiene igual (recupera panics, no tiene relación con el logging).
	r := gin.New()
	r.Use(middleware.StructuredLogger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     middleware.ParseAllowedOrigins(cfg.AllowedOriginsRaw),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Guarderia-Slug", "X-CSRF-Token", "X-Platform-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	srv.IniciarTareasProgramadas()
	if !srv.PushConfigurado() {
		applog.Warn("VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY no configuradas; las notificaciones push quedan deshabilitadas")
	}
	if !srv.StripeHabilitado() {
		applog.Info("STRIPE_SECRET_KEY no configurada; los pagos en línea quedan deshabilitados (el resto de la app no se ve afectado)")
	}
	if srv.PlatformAdminKey == "" {
		applog.Info("PLATFORM_ADMIN_KEY no configurada; las solicitudes de alta de guardería se pueden recibir pero no revisar/aprobar desde /plataforma")
	}
	// Se registra en ambos sentidos a propósito: sin esta línea, un chat de
	// soporte que no contesta solo se ve igual esté la IA prendida o
	// apagada, y no hay forma de saber desde los logs cuál de los dos casos
	// es. Ver Server.RAGSoporteHabilitado().
	if srv.RAGSoporteHabilitado() {
		applog.Info("Chat de soporte con IA habilitado (ANTHROPIC_API_KEY y VOYAGE_API_KEY configuradas)")
	} else {
		applog.Info("ANTHROPIC_API_KEY y/o VOYAGE_API_KEY sin configurar; el chat de soporte no responde solo y cada mensaje se avisa directo a la plataforma")
	}

	srv.RegisterRoutes(r)

	// PORT: Render (y la mayoría de plataformas tipo PaaS) asignan el puerto
	// ellos mismos vía esta variable -- 8099 fijo funcionaba porque en local
	// (Podman/Docker) nadie más decide el puerto, pero en Render el binario
	// tiene que escuchar en el que la plataforma indique, sea cual sea.
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8099"
	}

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		applog.Info("Sirviendo HTTPS directamente (TLS_CERT_FILE/TLS_KEY_FILE configurados)")
		if err := r.RunTLS(":"+puerto, cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			applog.Error("El servidor HTTPS se detuvo", err)
			os.Exit(1)
		}
		return
	}
	if err := r.Run(":" + puerto); err != nil {
		applog.Error("El servidor se detuvo", err)
		os.Exit(1)
	}
}

// fatal registra un error de arranque en el mismo formato JSON que el
// resto de la app (en vez de log.Fatal, que imprime texto plano) y termina
// el proceso -- estos errores pasan ANTES de levantar el servidor, así que
// nunca hay una petición real de por medio que loguear, pero el mensaje
// igual debe quedar en el mismo formato buscable que todo lo demás.
func fatal(msg string, err error) {
	applog.Error(msg, err)
	os.Exit(1)
}

// configurarPoolDB pone un tope explícito al pool de conexiones de cada
// *sql.DB (DB y DBAuth se configuran por separado, cada una con su propio
// pool). Sin esto, database/sql no limita cuántas conexiones abre: bajo
// carga (o con una fuga por un bug) podría abrir tantas como el servidor de
// Postgres le deje. Esto importa especialmente aquí porque DATABASE_URL/
// DATABASE_URL_AUTH apuntan a una instancia de Postgres COMPARTIDA con otro
// proyecto que ya paga esa instancia (ver render.yaml) -- sin tope, un pico
// de tráfico o un bug en Pasitos podría agotarle las conexiones disponibles
// también al otro proyecto, no solo al propio.
//
// Los números son deliberadamente conservadores para el tamaño actual (un
// piloto de una sola guardería, todavía sin tráfico real): hay margen de
// sobra para crecer antes de necesitar tocar esto. ConnMaxIdleTime bajo
// además ayuda a soltar conexiones ociosas rápido en vez de dejarlas
// reservadas sin usarlas.
func configurarPoolDB(db *sql.DB) {
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
}

// conectarServicios valida la configuración y abre las conexiones externas
// (AWS, Postgres) antes de levantar el router. Vive aquí, en cmd/server, y no
// en un func init() del paquete server: un init() que hace I/O y puede matar
// el proceso también correría al ejecutar "go test" sobre ese paquete.
func conectarServicios(cfg appconfig.Config) *server.Server {
	if cfg.AWSAccessKeyID == "" || cfg.AWSSecretAccessKey == "" {
		fatal("Credenciales de AWS no configuradas", nil)
	}

	if cfg.JWTSecret == "" {
		fatal("JWT_SECRET no configurada. Define esta variable de entorno con una clave secreta segura antes de iniciar el servidor", nil)
	}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		fatal("No se pudo cargar la configuración de AWS", err)
	}

	dbAuth, err := sql.Open("postgres", cfg.DatabaseURLAuth)
	if err != nil {
		fatal("Error conectando a DB Auth (DATABASE_URL_AUTH mal formada)", err)
	}
	configurarPoolDB(dbAuth)
	if err := dbAuth.Ping(); err != nil {
		fatal("Error conectando a DB Auth", err)
	}

	conexion, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fatal("Error conectando a la DB (DATABASE_URL mal formada)", err)
	}
	configurarPoolDB(conexion)

	if err := appdb.RunMigrations(conexion); err != nil {
		fatal("No se pudieron aplicar las migraciones", err)
	}

	if err = conexion.Ping(); err != nil {
		fatal("No se pudo conectar a la DB", err)
	}
	applog.Info("Conexión a Postgres exitosa")

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
	srv.PlatformAdminKey = cfg.PlatformAdminKey

	// Chat de soporte con IA -- ver Server.RAGSoporteHabilitado(). El
	// cliente de Anthropic solo se construye cuando hay clave; sin ella se
	// deja como el valor cero de anthropic.Client, que nunca se usa porque
	// RAGSoporteHabilitado() ya da false primero.
	srv.AnthropicAPIKey = cfg.AnthropicAPIKey
	srv.VoyageAPIKey = cfg.VoyageAPIKey
	if cfg.AnthropicAPIKey != "" {
		srv.AnthropicClient = anthropic.NewClient(option.WithAPIKey(cfg.AnthropicAPIKey))
	}

	return srv
}
