// Package config centraliza la lectura de variables de entorno. Antes esto
// vivía disperso entre func init() en main.go y variables globales sueltas en
// push.go (vapidPublicKey, etc).
package config

import "os"

type Config struct {
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	JWTSecret          string
	DatabaseURL        string
	DatabaseURLAuth    string
	AllowedOriginsRaw  string
	VapidPublicKey     string
	VapidPrivateKey    string
	VapidSubject       string
	// TLSCertFile/TLSKeyFile son opcionales: en producción (Render) el TLS lo
	// termina la plataforma y el proceso corre en HTTP plano — pero la cookie
	// de sesión es Secure (ver internal/middleware/auth.go), así que un
	// entorno sin ese borde HTTPS delante (ej. Docker/Podman local) necesita
	// que el propio binario sirva HTTPS para que el navegador la acepte. Si
	// cualquiera de las dos queda vacía, el servidor arranca en HTTP como
	// siempre.
	TLSCertFile string
	TLSKeyFile  string

	// Pagos móviles (Stripe Checkout) -- todas vacías por defecto, lo que
	// deja la función deshabilitada sin romper nada (igual que Vapid* para
	// push): no hay plan de activarla todavía, solo dejar el terreno listo.
	// Ver server.StripeHabilitado().
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	// StripeCurrency es el código ISO de 3 letras en minúsculas que Stripe
	// espera (ej. "mxn", "usd"). "mxn" por defecto -- la app está en
	// español de México y los montos ya se capturan así en /pagos.
	StripeCurrency string
	// FrontendURL arma las URLs de retorno del Checkout de Stripe
	// (éxito/cancelado) -- ej. "https://miguarderia.com". Solo hace falta
	// si StripeSecretKey está configurada.
	FrontendURL string

	// PlatformAdminKey protege /plataforma/solicitudes (revisar/aprobar
	// altas de guardería nueva) -- vacía por defecto, deja esas rutas
	// deshabilitadas. Ver middleware.RequirePlatformKey.
	PlatformAdminKey string
}

func Load() Config {
	return Config{
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:          os.Getenv("AWS_REGION"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		DatabaseURLAuth:    os.Getenv("DATABASE_URL_AUTH"),
		AllowedOriginsRaw:  os.Getenv("ALLOWED_ORIGINS"),
		VapidPublicKey:     os.Getenv("VAPID_PUBLIC_KEY"),
		VapidPrivateKey:    os.Getenv("VAPID_PRIVATE_KEY"),
		VapidSubject:       os.Getenv("VAPID_SUBJECT"),
		TLSCertFile:        os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:         os.Getenv("TLS_KEY_FILE"),

		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeCurrency:       monedaOr(os.Getenv("STRIPE_CURRENCY"), "mxn"),
		FrontendURL:          os.Getenv("FRONTEND_URL"),

		PlatformAdminKey: os.Getenv("PLATFORM_ADMIN_KEY"),
	}
}

func monedaOr(valor, porDefecto string) string {
	if valor == "" {
		return porDefecto
	}
	return valor
}
