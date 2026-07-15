package main

import (
	"errors"
	"testing"
)

// TestSave_valid teste qu'une note valide est sauvegardée sans erreur
func TestSave_valid(t *testing.T) {
	store := NewMemoryStore()
	note := &Note{
		ID:      "1",
		Title:   "Valid Note",
		Content: "This is a valid note",
		Tags:    []string{"test"},
	}

	err := store.Save(note)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Vérifier que la note a été sauvegardée
	retrieved, _ := store.Get("1")
	if retrieved.Title != "Valid Note" {
		t.Errorf("Expected title 'Valid Note', got %s", retrieved.Title)
	}
}

// TestSave_emptyTitle teste qu'une note avec titre vide retourne ErrValidation
func TestSave_emptyTitle(t *testing.T) {
	store := NewMemoryStore()
	note := &Note{
		ID:      "2",
		Title:   "   ", // Titre vide
		Content: "This should fail",
		Tags:    []string{},
	}

	err := store.Save(note)
	if err == nil {
		t.Error("Expected error for empty title, got nil")
	}

	if !errors.Is(err, ErrValidation) {
		t.Errorf("Expected ErrValidation, got %v", err)
	}
}

// TestSave_duplicate teste qu'une note avec un ID déjà existant retourne ErrDuplicate
func TestSave_duplicate(t *testing.T) {
	store := NewMemoryStore()

	// Première note
	note1 := &Note{
		ID:      "duplicate-test",
		Title:   "First Note",
		Content: "Content 1",
		Tags:    []string{},
	}
	err := store.Save(note1)
	if err != nil {
		t.Errorf("First save failed: %v", err)
	}

	// Deuxième note avec le même ID
	note2 := &Note{
		ID:      "duplicate-test",
		Title:   "Second Note",
		Content: "Content 2",
		Tags:    []string{},
	}
	err = store.Save(note2)
	if err == nil {
		t.Error("Expected error for duplicate ID, got nil")
	}

	if !errors.Is(err, ErrDuplicate) {
		t.Errorf("Expected ErrDuplicate, got %v", err)
	}
}

// TestGet_notFound teste que Get retourne ErrNotFound pour une note inexistante
func TestGet_notFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent note, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// Test bonus: TestAll vérifie que All() retourne toutes les notes
func TestAll(t *testing.T) {
	store := NewMemoryStore()

	note1 := &Note{ID: "1", Title: "Note 1", Content: "Content 1", Tags: []string{}}
	note2 := &Note{ID: "2", Title: "Note 2", Content: "Content 2", Tags: []string{}}
	note3 := &Note{ID: "3", Title: "Note 3", Content: "Content 3", Tags: []string{}}

	store.Save(note1)
	store.Save(note2)
	store.Save(note3)

	allNotes := store.All()
	if len(allNotes) != 3 {
		t.Errorf("Expected 3 notes, got %d", len(allNotes))
	}
}

// Test bonus: TestGet_afterSave vérifie que Get récupère correctement une note sauvegardée
func TestGet_afterSave(t *testing.T) {
	store := NewMemoryStore()

	note := &Note{
		ID:      "test-id",
		Title:   "Test Title",
		Content: "Test Content",
		Tags:    []string{"test", "go"},
	}

	store.Save(note)
	retrieved, err := store.Get("test-id")

	if err != nil {
		t.Errorf("Get failed after Save: %v", err)
	}

	if retrieved.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %s", retrieved.Title)
	}

	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}
}
