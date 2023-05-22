package must

// Panic if an err is given, else returns the T. This is a handy helper in a lot of
// situations!
func Panic[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}
