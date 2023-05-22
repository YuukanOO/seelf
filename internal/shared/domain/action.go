package domain

import "time"

// Action represents an action done by a user at a given time.
type Action[T ~string] struct {
	by T
	at time.Time
}

func NewAction[T ~string](by T) (a Action[T]) {
	a.by = by
	a.at = time.Now().UTC()
	return a
}

// Builds an action from both the user and the time, required when rehydrating
// the struct from the storage.
func ActionFrom[T ~string](by T, at time.Time) (a Action[T]) {
	a.by = by
	a.at = at
	return a
}

func (a Action[T]) By() T         { return a.by }
func (a Action[T]) At() time.Time { return a.at }
