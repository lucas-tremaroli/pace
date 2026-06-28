package epic

import "errors"

var (
	ErrEmptyID       = errors.New("epic id cannot be empty")
	ErrEmptyTitle    = errors.New("epic title cannot be empty")
	ErrInvalidStatus = errors.New("invalid epic status")
	ErrEpicNotFound  = errors.New("epic not found")
)
