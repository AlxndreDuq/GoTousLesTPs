package core

import (
	"fmt"
	"strings"
)

const (
	titleMaxLen   = 200
	contentMaxLen = 10000

	defaultLimit = 20
	maxLimit     = 100
)

// Repository is the persistence contract the service depends on.
// It is implemented by internal/store.
type Repository interface {
	Create(note Note) (Note, error)
	Get(id string) (Note, error)
	List(filter ListFilter) (ListResult, error)
	Update(id string, patch UpdateInput) (Note, error)
	Delete(id string) error
	Search(query string) ([]Note, error)
}

// Service holds the business logic for notes: validation and orchestration
// around the underlying Repository.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func isValidStatus(status string) bool {
	return status == StatusActive || status == StatusArchived
}

func (s *Service) CreateNote(input CreateInput) (Note, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Note{}, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if len(title) > titleMaxLen {
		return Note{}, fmt.Errorf("%w: title must be at most %d characters", ErrValidation, titleMaxLen)
	}
	if len(input.Content) > contentMaxLen {
		return Note{}, fmt.Errorf("%w: content must be at most %d characters", ErrValidation, contentMaxLen)
	}

	status := input.Status
	if status == "" {
		status = StatusActive
	} else if !isValidStatus(status) {
		return Note{}, fmt.Errorf("%w: status must be one of %q or %q", ErrValidation, StatusActive, StatusArchived)
	}

	return s.repo.Create(Note{
		Title:   title,
		Content: input.Content,
		Status:  status,
	})
}

func (s *Service) GetNote(id string) (Note, error) {
	return s.repo.Get(id)
}

func (s *Service) ListNotes(filter ListFilter) (ListResult, error) {
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return ListResult{}, fmt.Errorf("%w: status must be one of %q or %q", ErrValidation, StatusActive, StatusArchived)
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultLimit
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(filter)
}

func (s *Service) UpdateNote(id string, patch UpdateInput) (Note, error) {
	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if title == "" {
			return Note{}, fmt.Errorf("%w: title cannot be empty", ErrValidation)
		}
		if len(title) > titleMaxLen {
			return Note{}, fmt.Errorf("%w: title must be at most %d characters", ErrValidation, titleMaxLen)
		}
		patch.Title = &title
	}
	if patch.Content != nil && len(*patch.Content) > contentMaxLen {
		return Note{}, fmt.Errorf("%w: content must be at most %d characters", ErrValidation, contentMaxLen)
	}
	if patch.Status != nil && !isValidStatus(*patch.Status) {
		return Note{}, fmt.Errorf("%w: status must be one of %q or %q", ErrValidation, StatusActive, StatusArchived)
	}
	return s.repo.Update(id, patch)
}

func (s *Service) DeleteNote(id string) error {
	return s.repo.Delete(id)
}

func (s *Service) SearchNotes(query string) (ListResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return ListResult{}, fmt.Errorf("%w: q is required", ErrValidation)
	}
	notes, err := s.repo.Search(query)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Notes: notes, Total: len(notes)}, nil
}
