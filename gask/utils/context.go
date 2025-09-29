package utils

import (
	"context"
	"net/http"
)

// ContextKey type for context keys
type ContextKey string

const AuthContextKey ContextKey = "auth"

// SetAuthContext stores auth context in request context
func SetAuthContext(ctx context.Context, authCtx interface{}) context.Context {
	return context.WithValue(ctx, AuthContextKey, authCtx)
}

// GetAuthContext retrieves auth context from request context
func GetAuthContext(r *http.Request) interface{} {
	return r.Context().Value(AuthContextKey)
}

// GetAuthContextTyped retrieves auth context with type assertion
func GetAuthContextTyped(r *http.Request) (interface{}, bool) {
	authCtx := r.Context().Value(AuthContextKey)
	if authCtx != nil {
		return authCtx, true
	}
	return nil, false
}
