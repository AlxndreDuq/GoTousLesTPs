package core

import "errors"

// Sentinel errors mapped to HTTP status codes by the handlers layer.
var (
	ErrNotFound   = errors.New("note not found")
	ErrValidation = errors.New("validation error")
)
