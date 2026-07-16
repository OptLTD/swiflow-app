// Package util holds shared helpers for reusable library packages.
package support

import "github.com/google/uuid"

// NewID returns a new UUIDv7 string, falling back to v4 on error.
func NewID() string {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return u.String()
}
