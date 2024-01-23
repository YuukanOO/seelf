package domain

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/monad"
)

type contextKey string

const currentUserContextKey contextKey = "current-user"

// Attach the given UserID to the given context. Will be used everywhere when trying
// to determine which user is currently executing an action.
func WithUserID(ctx context.Context, uid UserID) context.Context {
	return context.WithValue(ctx, currentUserContextKey, uid)
}

// Retrieve the current user attached to the given context if any.
func CurrentUser(ctx context.Context) (m monad.Maybe[UserID]) {
	val := ctx.Value(currentUserContextKey)

	if val == nil {
		return m
	}

	m.Set(val.(UserID))

	return m
}
