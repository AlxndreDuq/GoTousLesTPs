package enrichment

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"mira-tp4/internal/core"
	"mira-tp4/internal/embedding"
)

// stopWords are excluded from tag extraction: too common to be useful as a
// tag, in either French or English since notes may be written in either.
var stopWords = map[string]struct{}{
	"les": {}, "des": {}, "une": {}, "un": {}, "que": {}, "qui": {}, "pour": {},
	"avec": {}, "dans": {}, "sur": {}, "sont": {}, "cette": {}, "ces": {},
	"pas": {}, "plus": {}, "mais": {}, "comme": {}, "leur": {}, "tout": {},
	"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {},
	"from": {}, "have": {}, "has": {}, "are": {}, "was": {}, "were": {},
	"your": {}, "you": {}, "not": {}, "all": {},
}

const (
	maxTags    = 5
	minTagLen  = 4
	maxSummary = 200
)

// NaiveEnricher derives tags/summary/score/embedding purely from the note's
// own text, with no external API call, so the pipeline runs fully offline.
type NaiveEnricher struct{}

func NewNaiveEnricher() NaiveEnricher {
	return NaiveEnricher{}
}

func (NaiveEnricher) Enrich(ctx context.Context, note core.Note) (core.EnrichmentResult, error) {
	if err := ctx.Err(); err != nil {
		return core.EnrichmentResult{}, err
	}

	text := note.Title + " " + note.Content
	tags := extractTags(text)
	summary := extractSummary(note.Content, note.Title)
	score := computeScore(note.Content, tags)
	vector := embedding.Embed(text)

	return core.EnrichmentResult{
		NoteID:    note.ID,
		Status:    core.EnrichmentDone,
		Tags:      tags,
		Summary:   summary,
		Score:     score,
		Embedding: vector,
	}, nil
}

func extractTags(text string) []string {
	counts := make(map[string]int)
	for _, word := range tokenizeWords(text) {
		if len(word) < minTagLen {
			continue
		}
		if _, stop := stopWords[word]; stop {
			continue
		}
		counts[word]++
	}

	words := make([]string, 0, len(counts))
	for w := range counts {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool {
		if counts[words[i]] != counts[words[j]] {
			return counts[words[i]] > counts[words[j]]
		}
		return words[i] < words[j] // stable tie-break for determinism
	})

	if len(words) > maxTags {
		words = words[:maxTags]
	}
	return words
}

func extractSummary(content, title string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return strings.TrimSpace(title)
	}

	end := strings.IndexAny(content, ".!?")
	sentence := content
	if end >= 0 {
		sentence = content[:end+1]
	}
	sentence = strings.TrimSpace(sentence)

	if len(sentence) > maxSummary {
		return truncateRunes(sentence, maxSummary)
	}
	return sentence
}

func computeScore(content string, tags []string) float64 {
	words := tokenizeWords(content)
	lengthScore := float64(len(words)) / 100.0
	if lengthScore > 1 {
		lengthScore = 1
	}
	tagScore := float64(len(tags)) / float64(maxTags)

	score := 0.6*lengthScore + 0.4*tagScore
	if score > 1 {
		score = 1
	}
	return score
}

func tokenizeWords(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	words := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			words = append(words, f)
		}
	}
	return words
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}
