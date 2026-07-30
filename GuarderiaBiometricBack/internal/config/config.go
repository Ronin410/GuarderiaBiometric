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
	}
}
