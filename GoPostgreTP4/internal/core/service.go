package core

import (
	"context"
	"fmt"
	"strings"
)

const (
	titleMaxLen   = 200
	contentMaxLen = 10000
	tagMaxLen     = 50
	tagsMaxCount  = 20

	defaultLimit = 20
	maxLimit     = 100
)

// Service holds the business logic for notes: validation and orchestration
// around the underlying Repository. Every successful create/update enqueues
// an enrichment job; the queue is optional (nil-safe) so tests can exercise
// the service without a running worker pool.
type Service struct {
	repo  Repository
	queue EnrichmentQueue
}

func NewService(repo Repository, queue EnrichmentQueue) *Service {
	return &Service{repo: repo, queue: queue}
}

func isValidStatus(status string) bool {
	return status == StatusActive || status == StatusArchived
}

func normalizeTags(tags []string) ([]string, error) {
	if len(tags) > tagsMaxCount {
		return nil, fmt.Errorf("%w: at most %d tags allowed", ErrValidation, tagsMaxCount)
	}
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if len(tag) > tagMaxLen {
			return nil, fmt.Errorf("%w: tag %q exceeds %d characters", ErrValidation, tag, tagMaxLen)
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized, nil
}

func (s *Service) enqueueEnrichment(noteID string) {
	if s.queue != nil {
		s.queue.Enqueue(noteID)
	}
}

func (s *Service) CreateNote(ctx context.Context, input CreateInput) (Note, error) {
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

	tags, err := normalizeTags(input.Tags)
	if err != nil {
		return Note{}, err
	}

	note, err := s.repo.Create(ctx, Note{
		Title:            title,
		Content:          input.Content,
		Status:           status,
		Tags:             tags,
		EnrichmentStatus: EnrichmentPending,
	})
	if err != nil {
		return Note{}, err
	}

	s.enqueueEnrichment(note.ID)
	return note, nil
}

func (s *Service) GetNote(ctx context.Context, id string) (Note, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) ListNotes(ctx context.Context, filter ListFilter) (ListResult, error) {
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
	return s.repo.List(ctx, filter)
}

func (s *Service) UpdateNote(ctx context.Context, id string, patch UpdateInput) (Note, error) {
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
	if patch.Tags != nil {
		tags, err := normalizeTags(*patch.Tags)
		if err != nil {
			return Note{}, err
		}
		patch.Tags = &tags
	}

	note, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Note{}, err
	}

	// Only title/content changes invalidate the previous enrichment
	// (tags/summary/score/embedding derive from them); the repository
	// resets enrichment_status to pending in the same transaction in that
	// case. A status- or tags-only patch leaves enrichment untouched.
	if patch.Title != nil || patch.Content != nil {
		s.enqueueEnrichment(note.ID)
	}
	return note, nil
}

func (s *Service) DeleteNote(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) SearchNotes(ctx context.Context, query string) (ListResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return ListResult{}, fmt.Errorf("%w: q is required", ErrValidation)
	}
	return s.repo.Search(ctx, query)
}
