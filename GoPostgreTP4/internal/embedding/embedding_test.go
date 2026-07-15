package embedding

import (
	"math"
	"testing"
)

func TestEmbed_Deterministic(t *testing.T) {
	a := Embed("Recette de pâtes à l'ail")
	b := Embed("Recette de pâtes à l'ail")

	if len(a) != Dimensions {
		t.Fatalf("len(a) = %d, want %d", len(a), Dimensions)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Embed is not deterministic at index %d: %v != %v", i, a[i], b[i])
		}
	}
}

func TestEmbed_Normalized(t *testing.T) {
	vec := Embed("un texte avec plusieurs mots distincts pour tester la norme")

	var sumSq float64
	for _, v := range vec {
		sumSq += float64(v) * float64(v)
	}
	norm := math.Sqrt(sumSq)

	if math.Abs(norm-1) > 1e-4 {
		t.Fatalf("||vec|| = %f, want ~1", norm)
	}
}

func TestEmbed_Empty(t *testing.T) {
	vec := Embed("")
	for i, v := range vec {
		if v != 0 {
			t.Fatalf("Embed(\"\")[%d] = %f, want 0", i, v)
		}
	}
}

func TestEmbed_SharedWordsAreCloser(t *testing.T) {
	base := Embed("chat noir qui dort sur le canapé")
	similar := Embed("chat noir qui dort sur le fauteuil")
	unrelated := Embed("rapport financier trimestriel du service comptabilité")

	if cosine(base, similar) <= cosine(base, unrelated) {
		t.Fatalf("expected texts sharing words to be closer: sim(base,similar)=%f, sim(base,unrelated)=%f",
			cosine(base, similar), cosine(base, unrelated))
	}
}

func cosine(a, b []float32) float64 {
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}
