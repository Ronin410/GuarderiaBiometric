package server

import (
	"testing"
	"time"
)

func TestHoyEnZonaLocal(t *testing.T) {
	loc := zonaMazatlan()

	casos := []struct {
		nombre   string
		entrada  time.Time
		esperado string
	}{
		{
			nombre:   "construido directo en la zona de la guardería",
			entrada:  time.Date(2026, 3, 15, 10, 0, 0, 0, loc),
			esperado: "2026-03-15",
		},
		{
			nombre: "el mismo instante recién antes de medianoche en Mazatlán sigue siendo el día anterior",
			// America/Mazatlan es UTC-7. 06:30 UTC del 15 de marzo son las 23:30
			// del 14 de marzo en Mazatlán — si el código formateara en UTC en vez
			// de convertir la zona primero (la clase de bug que ya se corrigió en
			// el frontend), esta prueba fallaría dando "2026-03-15".
			entrada:  time.Date(2026, 3, 15, 6, 30, 0, 0, time.UTC),
			esperado: "2026-03-14",
		},
		{
			nombre:   "un minuto después de medianoche en Mazatlán ya es el nuevo día",
			entrada:  time.Date(2026, 3, 15, 7, 1, 0, 0, time.UTC), // 00:01 en Mazatlán
			esperado: "2026-03-15",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := hoyEnZonaLocal(c.entrada)
			if got != c.esperado {
				t.Errorf("hoyEnZonaLocal(%v) = %q; esperado %q", c.entrada, got, c.esperado)
			}
		})
	}
}

func TestZonaMazatlanNoRegresaNil(t *testing.T) {
	if zonaMazatlan() == nil {
		t.Fatal("zonaMazatlan() no debe regresar nil; en el peor caso debe caer a UTC")
	}
}
