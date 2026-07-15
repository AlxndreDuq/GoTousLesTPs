package search

import (
	"strings"

	"mira/internal/notes"
)

// Search retourne les notes dont le titre ou le contenu contient query,
// insensible à la casse.
func Search(all []notes.Note, query string) []notes.Note {
	q := strings.ToLower(query)

	var result []notes.Note
	for _, n := range all {
		if strings.Contains(strings.ToLower(n.Title), q) ||
			strings.Contains(strings.ToLower(n.Content), q) {
			result = append(result, n)
		}
	}

	return result
}
