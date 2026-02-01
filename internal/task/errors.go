package task

import "errors"

// Error definitions for task validation and operations
var (
	ErrEmptyTitle    = errors.New("task title cannot be empty")
	ErrInvalidStatus = errors.New("invalid task status")
	ErrInvalidLink   = errors.New("invalid link: must be a valid URL (e.g. https://example.com)")
)
