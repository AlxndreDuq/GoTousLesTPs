package enrichment

import (
	"context"
	"testing"

	"mira-tp4/internal/core"
)

func TestNaiveEnricher_Enrich(t *testing.T) {
	e := NewNaiveEnricher()
	note := core.Note{
		ID:      "note-1",
		Title:   "Recette pâtes carbonara",
		Content: "Faire cuire les pâtes. Ajouter lardons, œufs et parmesan. Ne pas ajouter de crème.",
	}

	result, err := e.Enrich(context.Background(), note)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}

	if result.Status != core.EnrichmentDone {
		t.Fatalf("Status = %q, want %q", result.Status, core.EnrichmentDone)
	}
	if result.NoteID != note.ID {
		t.Fatalf("NoteID = %q, want %q", result.NoteID, note.ID)
	}
	if len(result.Tags) == 0 {
		t.Fatal("expected at least one tag")
	}
	if result.Summary == "" {
		t.Fatal("expected a non-empty summary")
	}
	if result.Score < 0 || result.Score > 1 {
		t.Fatalf("Score = %f, want in [0,1]", result.Score)
	}
	if len(result.Embedding) == 0 {
		t.Fatal("expected a non-empty embedding")
	}
}

func TestNaiveEnricher_EmptyContentFallsBackToTitle(t *testing.T) {
	e := NewNaiveEnricher()
	note := core.Note{ID: "note-2", Title: "Idée sans détails"}

	result, err := e.Enrich(context.Background(), note)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result.Summary != note.Title {
		t.Fatalf("Summary = %q, want %q", result.Summary, note.Title)
	}
}

func TestNaiveEnricher_RespectsCancelledContext(t *testing.T) {
	e := NewNaiveEnricher()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.Enrich(ctx, core.Note{ID: "note-3", Title: "x"})
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
}

func TestExtractTags_ExcludesStopWordsAndShortWords(t *testing.T) {
	tags := extractTags("Les pâtes sont pour tous avec du fromage et un peu de crème")
	for _, tag := range tags {
		if _, stop := stopWords[tag]; stop {
			t.Fatalf("tag %q should have been filtered out as a stop word", tag)
		}
		if len(tag) < minTagLen {
			t.Fatalf("tag %q is shorter than minTagLen=%d", tag, minTagLen)
		}
	}
}
