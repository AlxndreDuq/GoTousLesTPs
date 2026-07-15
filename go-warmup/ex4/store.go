package main

import (
	"strings"
	"sync"
)

// NoteStore est l'interface pour stocker et récupérer des notes
type NoteStore interface {
	Save(n *Note) error
	Get(id string) (*Note, error)
	All() []*Note
}

// MemoryStore implémente NoteStore avec une map en mémoire
type MemoryStore struct {
	mu    sync.Mutex
	notes map[string]*Note
}

// NewMemoryStore crée une nouvelle instance de MemoryStore
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		notes: make(map[string]*Note),
	}
}

// Save sauvegarde une note dans le store
func (ms *MemoryStore) Save(n *Note) error {
	// Valider que le titre n'est pas vide
	if strings.TrimSpace(n.Title) == "" {
		return ErrValidation
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Vérifier que l'ID n'existe pas déjà
	if _, exists := ms.notes[n.ID]; exists {
		return ErrDuplicate
	}

	// Sauvegarder la note
	ms.notes[n.ID] = n
	return nil
}

// Get récupère une note par ID
func (ms *MemoryStore) Get(id string) (*Note, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	note, exists := ms.notes[id]
	if !exists {
		return nil, ErrNotFound
	}

	return note, nil
}

// All retourne toutes les notes du store
func (ms *MemoryStore) All() []*Note {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	result := make([]*Note, 0, len(ms.notes))
	for _, note := range ms.notes {
		result = append(result, note)
	}
	return result
}
