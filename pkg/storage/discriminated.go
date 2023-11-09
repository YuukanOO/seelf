package storage

type (
	// Represents an extensible type known through a discriminator value when implementing
	// the Scanner interface as no sense since we don't know which type is used in our entity.
	Discriminated interface {
		Discriminator() string
	}

	// Function used to map from a raw value to a discriminated type.
	DiscriminatedMapperFunc[T Discriminated] func(string) (T, error)

	// Mapper struct to be able to rehydrate discriminated types.
	DiscriminatedMapper[T Discriminated] map[string]DiscriminatedMapperFunc[T]
)

// Builds a new mapper configuration to hold a discriminated type.
func NewDiscriminatedMapper[T Discriminated]() DiscriminatedMapper[T] {
	return make(DiscriminatedMapper[T])
}

// Register a new concrete type available to the mapper.
func (m DiscriminatedMapper[T]) Register(concreteType T, mapper DiscriminatedMapperFunc[T]) {
	discriminator := concreteType.Discriminator()

	// Check for duplicate registrations, should panic because it's a dev error.
	if _, found := m[discriminator]; found {
		panic("duplicate concrete type registered for " + discriminator)
	}

	m[discriminator] = mapper
}

// Rehydrate a discriminated type from a raw value.
func (m DiscriminatedMapper[T]) From(discriminator, value string) (T, error) {
	mapper, found := m[discriminator]

	if !found {
		var t T
		return t, ErrCouldNotUnmarshalGivenType
	}

	return mapper(value)
}
