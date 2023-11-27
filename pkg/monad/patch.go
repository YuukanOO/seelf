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

// Try to get the inner optional value for this Patch structure. The boolean returns
// if the patch has been set so the returned value may represent a nil value.
func (p Patch[T]) TryGet() (Maybe[T], bool) {
	return p.Maybe, p.isSet
}

// Implements the UnmarshalJSON interface.
func (p *Patch[T]) UnmarshalJSON(data []byte) error {
	// If we're here, it means the value has been set
	p.isSet = true

	return p.Maybe.UnmarshalJSON(data)
}
