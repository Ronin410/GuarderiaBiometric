// Package email manda avisos por correo -- de momento solo se usa para
// avisarle al dueño de la plataforma cuando entra un mensaje nuevo al chat
// de soporte (ver internal/server/chat_soporte.go). A propósito usa
// net/smtp (sin dependencias nuevas ni una cuenta en un servicio de
// terceros) en vez de un proveedor de correo transaccional -- "no le quiero
// agregar ahorita un monitoreo tercero" ya fue la instrucción para el resto
// de observabilidad, y esto sigue el mismo criterio: un remitente SMTP que
// el dueño de la plataforma ya controla (su propio Gmail, por ejemplo)
// en vez de una cuenta nueva en otro servicio.
package email

import (
	"fmt"
	"net/smtp"
)

// Config trae las credenciales SMTP -- vacía por defecto, lo que deja la
// función deshabilitada sin romper nada (mismo criterio que
// Server.PushConfigurado()/StripeHabilitado(): sin configurar, sencillamente
// no está disponible en vez de fallar).
type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// Configurado indica si hay credenciales SMTP suficientes para mandar
// correo.
func (c Config) Configurado() bool {
	return c.Host != "" && c.Port != "" && c.User != "" && c.Pass != ""
}

// Enviar manda un correo de texto plano. Si Config no está configurado, no
// hace nada y regresa nil -- el caller no necesita revisar Configurado()
// aparte, igual que PushConfigurado() no le hace falta a cada llamada de
// notificar*.
func (c Config) Enviar(destinatario, asunto, cuerpo string) error {
	if !c.Configurado() {
		return nil
	}
	if destinatario == "" {
		return nil
	}

	remitente := c.From
	if remitente == "" {
		remitente = c.User
	}

	addr := fmt.Sprintf("%s:%s", c.Host, c.Port)
	auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)

	// Cabeceras mínimas -- SendMail hace STARTTLS solo si el servidor lo
	// ofrece (Gmail y la gran mayoría de proveedores en el puerto 587), así
	// que no hace falta manejar TLS a mano aquí.
	msg := []byte(
		"From: " + remitente + "\r\n" +
			"To: " + destinatario + "\r\n" +
			"Subject: " + asunto + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			cuerpo + "\r\n",
	)

	return smtp.SendMail(addr, auth, remitente, []string{destinatario}, msg)
}
