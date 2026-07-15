package main

import "errors"

var (
	ErrDuplicate  = errors.New("note already exists")
	ErrNotFound   = errors.New("note not found")
	ErrValidation = errors.New("validation error: title cannot be empty")
)
