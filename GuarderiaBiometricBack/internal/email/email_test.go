package email

import "testing"

func TestConfigurado(t *testing.T) {
	casos := []struct {
		nombre string
		cfg    Config
		quiere bool
	}{
		{"vacío", Config{}, false},
		{"completo", Config{Host: "smtp.gmail.com", Port: "587", User: "a@b.com", Pass: "clave"}, true},
		{"sin contraseña", Config{Host: "smtp.gmail.com", Port: "587", User: "a@b.com"}, false},
		{"sin usuario", Config{Host: "smtp.gmail.com", Port: "587", Pass: "clave"}, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := c.cfg.Configurado(); got != c.quiere {
				t.Errorf("Configurado() = %v; se esperaba %v", got, c.quiere)
			}
		})
	}
}

// TestEnviarSinConfigurar -- sin credenciales, Enviar no debe intentar
// conectarse a ningún servidor (regresaría un error de red, no nil) --
// mismo criterio que PushConfigurado()/StripeHabilitado(): sin configurar,
// la función simplemente no está disponible.
func TestEnviarSinConfigurar(t *testing.T) {
	var cfg Config
	if err := cfg.Enviar("alguien@ejemplo.com", "Asunto", "Cuerpo"); err != nil {
		t.Errorf("Enviar() sin configurar = %v; se esperaba nil", err)
	}
}

func TestEnviarSinDestinatario(t *testing.T) {
	cfg := Config{Host: "smtp.gmail.com", Port: "587", User: "a@b.com", Pass: "clave"}
	if err := cfg.Enviar("", "Asunto", "Cuerpo"); err != nil {
		t.Errorf("Enviar() sin destinatario = %v; se esperaba nil", err)
	}
}
