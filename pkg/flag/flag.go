package flag

type Flag interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Check if the given flag value has one of the given flags.
func IsSet[T Flag](value, check T) bool {
	return value&check == check
}
