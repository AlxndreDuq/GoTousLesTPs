package core

import "time"

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// Note is the domain representation of a note.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInput carries the fields accepted when creating a note.
type CreateInput struct {
	Title   string
	Content string
	Status  string
}

// UpdateInput carries the fields accepted when partially updating a note.
// A nil pointer means "field not provided" so it is left untouched.
type UpdateInput struct {
	Title   *string
	Content *string
	Status  *string
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
