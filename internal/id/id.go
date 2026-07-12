// Package id generates time-ordered identifiers (UUIDv7).
package id

import "github.com/google/uuid"

// New returns a new UUIDv7 string, falling back to v4 on error.
func New() string {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return u.String()
}
