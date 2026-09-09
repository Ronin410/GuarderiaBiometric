package server

import "testing"

func TestExtraerKeyS3(t *testing.T) {
	casos := []struct {
		nombre   string
		entrada  string
		esperado string
	}{
		{
			nombre:   "key nueva (formato posterior al cierre del bucket público)",
			entrada:  "guarderia_1/hijo_3/2026-07-30_153000_foto.jpg",
			esperado: "guarderia_1/hijo_3/2026-07-30_153000_foto.jpg",
		},
		{
			nombre:   "URL pública completa (fotos subidas antes del cierre del bucket)",
			entrada:  "https://biosafe-storage-fotos.s3.us-east-1.amazonaws.com/guarderia_1/hijo_3/2026-07-30_153000_foto.jpg",
			esperado: "guarderia_1/hijo_3/2026-07-30_153000_foto.jpg",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := extraerKeyS3(c.entrada)
			if got != c.esperado {
				t.Errorf("extraerKeyS3(%q) = %q; esperado %q", c.entrada, got, c.esperado)
			}
		})
	}
}
