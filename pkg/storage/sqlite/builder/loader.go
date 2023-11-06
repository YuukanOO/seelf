package builder

import "context"

type (
	// Represents an element which can fetch additional data from a T parent entity.
	// Dataloaders will be called with a list of keys to fetch as a whole to avoid
	// N+1 queries.
	Dataloader[T any] interface {
		ExtractKey(T) string                                   // Extract the key from the parent data. Will be given to the Fetch function afterwards.
		Fetch(Executor, context.Context, KeyedResult[T]) error // Fetch related data represented by this dataloader.
	}

	dataLoader[T any] struct {
		extractor func(T) string
		fetcher   func(Executor, context.Context, KeyedResult[T]) error
	}
)

// Builds up a new dataloader.
func NewDataloader[T any](
	extractor func(T) string,
	fetcher func(Executor, context.Context, KeyedResult[T]) error,
) Dataloader[T] {
	return &dataLoader[T]{extractor, fetcher}
}

func (l *dataLoader[T]) ExtractKey(data T) string {
	return l.extractor(data)
}

func (l *dataLoader[T]) Fetch(ex Executor, ctx context.Context, result KeyedResult[T]) error {
	return l.fetcher(ex, ctx, result)
}
