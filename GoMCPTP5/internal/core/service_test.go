package core_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"mira-tp4/internal/core"
)

// fakeRepo is a minimal in-memory core.Repository for exercising Service's
// validation/orchestration logic in isolation from Postgres.
type fakeRepo struct {
	mu     sync.Mutex
	notes  map[string]core.Note
	nextID int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{notes: make(map[string]core.Note)}
}

func (r *fakeRepo) Create(_ context.Context, note core.Note) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	note.ID = strconv.Itoa(r.nextID)
	note.CreatedAt = time.Now()
	note.UpdatedAt = note.CreatedAt
	r.notes[note.ID] = note
	return note, nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}
	return note, nil
}

func (r *fakeRepo) List(_ context.Context, _ core.ListFilter) (core.ListResult, error) {
	return core.ListResult{}, nil
}

func (r *fakeRepo) Update(_ context.Context, id string, patch core.UpdateInput) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	note, ok := r.notes[id]
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
	if patch.Tags != nil {
		note.Tags = *patch.Tags
	}
	r.notes[id] = note
	return note, nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.notes[id]; !ok {
		return core.ErrNotFound
	}
	delete(r.notes, id)
	return nil
}

func (r *fakeRepo) Search(_ context.Context, _ string) (core.ListResult, error) {
	return core.ListResult{}, nil
}

func (r *fakeRepo) SaveEnrichment(_ context.Context, _ core.EnrichmentResult) error {
	return nil
}

// fakeQueue records every note id it was asked to enqueue.
type fakeQueue struct {
	mu  sync.Mutex
	ids []string
}

func (q *fakeQueue) Enqueue(noteID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.ids = append(q.ids, noteID)
}

func (q *fakeQueue) enqueued() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]string(nil), q.ids...)
}

func TestCreateNote_EnqueuesEnrichment(t *testing.T) {
	repo := newFakeRepo()
	queue := &fakeQueue{}
	svc := core.NewService(repo, queue)

	note, err := svc.CreateNote(context.Background(), core.CreateInput{Title: "Courses"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}

	enqueued := queue.enqueued()
	if len(enqueued) != 1 || enqueued[0] != note.ID {
		t.Fatalf("enqueued = %v, want [%q]", enqueued, note.ID)
	}
}

func TestCreateNote_ValidationErrorDoesNotEnqueue(t *testing.T) {
	repo := newFakeRepo()
	queue := &fakeQueue{}
	svc := core.NewService(repo, queue)

	_, err := svc.CreateNote(context.Background(), core.CreateInput{Title: "   "})
	if !errors.Is(err, core.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
	if len(queue.enqueued()) != 0 {
		t.Fatalf("expected no enqueued job, got %v", queue.enqueued())
	}
}

func TestUpdateNote_StatusOnlyPatchDoesNotEnqueue(t *testing.T) {
	repo := newFakeRepo()
	queue := &fakeQueue{}
	svc := core.NewService(repo, queue)

	note, err := svc.CreateNote(context.Background(), core.CreateInput{Title: "Courses"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	queue.mu.Lock()
	queue.ids = nil // reset: only interested in what Update triggers
	queue.mu.Unlock()

	archived := core.StatusArchived
	if _, err := svc.UpdateNote(context.Background(), note.ID, core.UpdateInput{Status: &archived}); err != nil {
		t.Fatalf("UpdateNote() error = %v", err)
	}

	if len(queue.enqueued()) != 0 {
		t.Fatalf("expected status-only patch not to enqueue enrichment, got %v", queue.enqueued())
	}
}

func TestUpdateNote_ContentPatchEnqueues(t *testing.T) {
	repo := newFakeRepo()
	queue := &fakeQueue{}
	svc := core.NewService(repo, queue)

	note, err := svc.CreateNote(context.Background(), core.CreateInput{Title: "Courses"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	queue.mu.Lock()
	queue.ids = nil
	queue.mu.Unlock()

	content := "Lait, oeufs"
	if _, err := svc.UpdateNote(context.Background(), note.ID, core.UpdateInput{Content: &content}); err != nil {
		t.Fatalf("UpdateNote() error = %v", err)
	}

	enqueued := queue.enqueued()
	if len(enqueued) != 1 || enqueued[0] != note.ID {
		t.Fatalf("enqueued = %v, want [%q]", enqueued, note.ID)
	}
}

func TestSearchNotes_EmptyQueryIsValidationError(t *testing.T) {
	svc := core.NewService(newFakeRepo(), nil)

	_, err := svc.SearchNotes(context.Background(), "   ")
	if !errors.Is(err, core.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateNote_TooManyTagsIsValidationError(t *testing.T) {
	svc := core.NewService(newFakeRepo(), nil)

	tags := make([]string, 21)
	for i := range tags {
		tags[i] = strconv.Itoa(i)
	}

	_, err := svc.CreateNote(context.Background(), core.CreateInput{Title: "x", Tags: tags})
	if !errors.Is(err, core.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}
