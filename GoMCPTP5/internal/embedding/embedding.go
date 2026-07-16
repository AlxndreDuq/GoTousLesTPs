// Package embedding derives a deterministic pseudo-embedding from text,
// with no external model or network call involved. It exists so the rest of
// the pipeline (enrichment, hybrid search) can exercise a real pgvector
// column, index and similarity search end to end without depending on a
// third-party embeddings API or key.
package embedding

import (
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// Dimensions is the size of vectors produced by Embed; it must match the
// VECTOR(64) column defined in migrations/0001_init.sql.
const Dimensions = 64

// Embed hashes each token of text into a bucket of a fixed-size vector, then
// L2-normalizes it. Texts sharing many words end up with a smaller cosine
// distance, which is enough to demonstrate vector similarity search.
func Embed(text string) []float32 {
	vec := make([]float32, Dimensions)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vec
	}

	for _, tok := range tokens {
		idx := hashString(tok) % Dimensions
		sign := float32(1)
		if hashString(tok+"#sign")%2 == 0 {
			sign = -1
		}
		vec[idx] += sign
	}

	normalize(vec)
	return vec
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func normalize(vec []float32) {
	var sumSq float64
	for _, v := range vec {
		sumSq += float64(v) * float64(v)
	}
	if sumSq == 0 {
		return
	}
	norm := float32(math.Sqrt(sumSq))
	for i := range vec {
		vec[i] /= norm
	}
}

func tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) >= 2 {
			tokens = append(tokens, f)
		}
	}
	return tokens
}
