package model

import "errors"

var (
	// ErrNotFound is returned when an entity is required to exist and does not. 
	ErrNotFound = errors.New("entity was not found")
)
