package ia

import "testing"

func TestVectorLiteral(t *testing.T) {
	tests := []struct {
		nombre    string
		embedding []float32
		esperado  string
	}{
		{"vacío", []float32{}, "[]"},
		{"un valor", []float32{0.5}, "[0.5]"},
		{"varios valores", []float32{0.1, -0.25, 1, 0}, "[0.1,-0.25,1,0]"},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			resultado := VectorLiteral(tt.embedding)
			if resultado != tt.esperado {
				t.Errorf("VectorLiteral(%v) = %q; esperado %q", tt.embedding, resultado, tt.esperado)
			}
		})
	}
}

func TestGenerarEmbeddingsSinTextos(t *testing.T) {
	// Sin textos de entrada no debe intentar llamar a la API -- regresa
	// nil sin error, sin necesitar una API key válida para esta rama.
	embeddings, err := GenerarEmbeddings("", nil, "query")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if embeddings != nil {
		t.Errorf("se esperaba nil, se obtuvo %v", embeddings)
	}
}
