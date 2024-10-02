package storage

import "context"

// Factory to retrieve a transactional context to make sure everything is atomic.
type UnitOfWorkFactory interface {
	Create(context.Context, func(context.Context) error) error
}
