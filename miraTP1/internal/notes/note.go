package notes

import "time"

// Note représente une note personnelle stockée par mira.
type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// NewNote crée une nouvelle note horodatée à l'instant présent.
func NewNote(title, content string) Note {
	now := time.Now()
	return Note{
		ID:        now.UnixNano(),
		Title:     title,
		Content:   content,
		CreatedAt: now,
	}
}
