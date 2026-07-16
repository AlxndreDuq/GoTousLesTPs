package core

import "time"

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

const (
	EnrichmentPending = "pending"
	EnrichmentDone    = "done"
	EnrichmentFailed  = "failed"
)

// Note is the domain representation of a note.
type Note struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Status           string    `json:"status"`
	Tags             []string  `json:"tags"`
	EnrichmentStatus string    `json:"enrichment_status"`
	Summary          string    `json:"summary,omitempty"`
	Score            *float64  `json:"score,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateInput carries the fields accepted when creating a note.
type CreateInput struct {
	Title   string
	Content string
	Status  string
	Tags    []string
}

// UpdateInput carries the fields accepted when partially updating a note.
// A nil pointer means "field not provided" so it is left untouched; the
// same convention applies to Tags via a pointer to a slice, since a nil
// slice and an empty (clearing) slice must be distinguishable.
type UpdateInput struct {
	Title   *string
	Content *string
	Status  *string
	Tags    *[]string
}

// ListFilter carries the filtering/pagination options for List.
type ListFilter struct {
	Status string
	Limit  int
	Offset int
}

// ListResult carries a page of notes plus the total count matching the filter.
type ListResult struct {
	Notes []Note
	Total int
}

// EnrichmentResult carries the outcome of an enrichment task, ready to be
// persisted by Repository.SaveEnrichment.
type EnrichmentResult struct {
	NoteID    string
	Status    string // EnrichmentDone or EnrichmentFailed
	Tags      []string
	Summary   string
	Score     float64
	Embedding []float32
}
