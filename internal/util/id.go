// Package util holds small shared helpers used across Mira.
package util

import "github.com/google/uuid"

// NewID returns a new UUIDv7 string, falling back to v4 on error.
func NewID() string {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return u.String()
}
