// Package tenant carries the active tenant id on context.Context.
package tenant

import "context"

// DefaultID is the single-tenant / Desktop LocalMode tenant.
const DefaultID = "default"

type ctxKey struct{}

// WithID returns a child context carrying tid (empty becomes DefaultID).
func WithID(ctx context.Context, tid string) context.Context {
	if tid == "" {
		tid = DefaultID
	}
	return context.WithValue(ctx, ctxKey{}, tid)
}

// ID returns the tenant id from ctx, or DefaultID when unset.
func ID(ctx context.Context) string {
	if ctx == nil {
		return DefaultID
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok && v != "" {
		return v
	}
	return DefaultID
}
