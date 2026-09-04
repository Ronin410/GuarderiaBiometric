package main

import (
	"strings"
	"testing"
)

// TestExtraerFragmentosManuales corre extraerFragmentos contra los manuales
// reales del repo (no un HTML de prueba inventado) -- así un cambio de
// estructura en manual.html/manual-papa.html que rompa el parseo (por
// ejemplo, renombrar la clase "manual" de <article>) se detecta aquí en vez
// de solo notarse la próxima vez que alguien reindexe a mano.
func TestExtraerFragmentosManuales(t *testing.T) {
	for _, ruta := range []string{
		"../../../GuarderiaBiometricFront/public/manual.html",
		"../../../GuarderiaBiometricFront/public/manual-papa.html",
	} {
		t.Run(ruta, func(t *testing.T) {
			fragmentos, err := extraerFragmentos(ruta)
			if err != nil {
				t.Fatalf("extraerFragmentos: %v", err)
			}
			if len(fragmentos) < 10 {
				t.Fatalf("se esperaban al menos 10 fragmentos, se encontraron %d -- ¿cambió la estructura del HTML?", len(fragmentos))
			}

			nombreArchivo := ruta[strings.LastIndex(ruta, "/")+1:]
			for _, f := range fragmentos {
				if f.Contenido == "" {
					t.Errorf("fragmento con contenido vacío (fuente: %q)", f.Fuente)
				}
				if !strings.HasPrefix(f.Fuente, nombreArchivo) {
					t.Errorf("fuente %q no empieza con el nombre del archivo %q", f.Fuente, nombreArchivo)
				}
				// Sin restos de HTML -- si aparece un "<" es que alguna
				// etiqueta no se limpió bien.
				if strings.Contains(f.Contenido, "<") {
					t.Errorf("el contenido parece traer HTML sin limpiar: %q", f.Contenido)
				}
			}
		})
	}
}
