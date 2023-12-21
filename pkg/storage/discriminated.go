package storage

type (
	// Function used to map from a raw value to a discriminated type.
	DiscriminatedMapperFunc[T any] func(string) (T, error)

	// Function used to extract a discriminator from a value.
	DiscriminatorFunc[T any] func(T) string

	// Mapper struct to be able to rehydrate discriminated types.
	// Discriminated types represents an extensible type known through a discriminator value. Building a
	// specific mapper makes it easy to reconstruct a specific type from a discriminator and raw data.
	// Without it, retrieving dynamic types from the database is a nightmare. With this solution though,
	// it's not that bad ;)
	DiscriminatedMapper[T any] struct {
		known     map[string]DiscriminatedMapperFunc[T]
		extractor DiscriminatorFunc[T]
	}
)

// Builds a new mapper configuration to hold a discriminated type.
func NewDiscriminatedMapper[T any](
	extractor DiscriminatorFunc[T],
) *DiscriminatedMapper[T] {
	return &DiscriminatedMapper[T]{
		known:     make(map[string]DiscriminatedMapperFunc[T]),
		extractor: extractor,
	}
}

// Register a new concrete type available to the mapper.
func (m *DiscriminatedMapper[T]) Register(concreteType T, mapper DiscriminatedMapperFunc[T]) {
	discriminator := m.extractor(concreteType)

	// Check for duplicate registrations, should panic because it's a dev error.
	if _, found := m.known[discriminator]; found {
		panic("duplicate concrete type registered for " + discriminator)
	}

	m.known[discriminator] = mapper
}

// Rehydrate a discriminated type from a raw value.
func (m *DiscriminatedMapper[T]) From(discriminator, value string) (T, error) {
	mapper, found := m.known[discriminator]

	if !found {
		var t T
		return t, ErrCouldNotUnmarshalGivenType
	}

	return mapper(value)
}
