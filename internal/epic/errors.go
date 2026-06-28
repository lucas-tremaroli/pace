package epic

import "errors"

var (
	ErrEmptyTitle    = errors.New("epic title cannot be empty")
	ErrInvalidStatus = errors.New("invalid epic status")
	ErrEpicNotFound  = errors.New("epic not found")
)
