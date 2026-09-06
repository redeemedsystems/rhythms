package store

import "errors"

// ErrNotFound is wrapped by repo methods when a lookup or mutation targets a
// row that doesn't exist.
var ErrNotFound = errors.New("not found")
