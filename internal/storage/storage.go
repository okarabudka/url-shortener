// Package storage defines the errors every storage implementation returns.
package storage

import "errors"

// Errors returned by storage implementations, wrapped with the failing operation.
var (
	ErrURLNotFound = errors.New("url not found")
	ErrURLExists   = errors.New("url exists")
)
