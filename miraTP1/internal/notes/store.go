package notes

// NoteStore définit les opérations de persistance des notes.
type NoteStore interface {
	Add(note Note) error
	List(limit int) ([]Note, error)
	All() ([]Note, error)
}
