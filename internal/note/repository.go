package note

// NoteRepository abstracts storage operations needed by the note service.
// *storage.DB satisfies this interface; nil is valid when no DB is configured.
type NoteRepository interface {
	Close() error
	GetTasksForNote(noteFilename string) ([]string, error)
	RemoveAllNoteLinks(noteFilename string) error
}
