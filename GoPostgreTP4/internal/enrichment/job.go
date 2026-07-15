package enrichment

// Job is a unit of work published to the internal channel: "go enrich this
// note". It intentionally carries only an ID, not the note itself, so the
// worker always re-reads the current row before enriching it.
type Job struct {
	NoteID string
}
