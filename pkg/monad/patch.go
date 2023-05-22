package monad

// Type used to patch data. Those values could be set, nil or have a value.
type Patch[T any] struct {
	Maybe[T]
	isSet bool
}

// Builds a new patch with the given value.
func PatchValue[T any](value T) (p Patch[T]) {
	p.isSet = true
	p.Maybe = p.Maybe.WithValue(value)
	return p
}

// Builds a nil patch.
func Nil[T any]() (p Patch[T]) {
	p.isSet = true
	return p
}

func (p Patch[T]) IsSet() bool { return p.isSet }
func (p Patch[T]) IsNil() bool { return p.isSet && !p.hasValue }

// Implements the UnmarshalJSON interface.
func (p *Patch[T]) UnmarshalJSON(data []byte) error {
	// If we're here, it means the value has been set
	p.isSet = true

	return p.Maybe.UnmarshalJSON(data)
}
