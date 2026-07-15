package store

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mira-tp2/internal/core"
)

// MemoryStore is an in-memory, concurrency-safe implementation of
// core.Repository backed by a map protected by a mutex.
type MemoryStore struct {
	mu     sync.RWMutex
	notes  map[string]core.Note
	nextID uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		notes: make(map[string]core.Note),
	}
}

func (s *MemoryStore) Create(note core.Note) (core.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	now := time.Now().UTC()
	note.ID = strconv.FormatUint(s.nextID, 10)
	note.CreatedAt = now
	note.UpdatedAt = now
	s.notes[note.ID] = note
	return note, nil
}

func (s *MemoryStore) Get(id string) (core.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	note, ok := s.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}
	return note, nil
}

func (s *MemoryStore) List(filter core.ListFilter) (core.ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]core.Note, 0, len(s.notes))
	for _, note := range s.notes {
		if filter.Status != "" && note.Status != filter.Status {
			continue
		}
		matched = append(matched, note)
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.Before(matched[j].CreatedAt)
	})

	total := len(matched)
	start := filter.Offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}

	return core.ListResult{Notes: matched[start:end], Total: total}, nil
}

func (s *MemoryStore) Update(id string, patch core.UpdateInput) (core.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}

	if patch.Title != nil {
		note.Title = *patch.Title
	}
	if patch.Content != nil {
		note.Content = *patch.Content
	}
	if patch.Status != nil {
		note.Status = *patch.Status
	}
	note.UpdatedAt = time.Now().UTC()

	s.notes[id] = note
	return note, nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[id]; !ok {
		return core.ErrNotFound
	}
	delete(s.notes, id)
	return nil
}

func (s *MemoryStore) Search(query string) ([]core.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	matched := make([]core.Note, 0)
	for _, note := range s.notes {
		if strings.Contains(strings.ToLower(note.Title), query) ||
			strings.Contains(strings.ToLower(note.Content), query) {
			matched = append(matched, note)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.Before(matched[j].CreatedAt)
	})
	return matched, nil
}
