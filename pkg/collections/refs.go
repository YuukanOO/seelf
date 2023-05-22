package collections

// Convert an array of item to an array of pointer of items. This is sometimes needed
// when you want to pass a slice of items to a function that expects a slice of pointers
func ToPointers[T any](items []T) []*T {
	results := make([]*T, len(items))

	for idx := range items {
		// Needed or else it will always point to the last item being iterated
		item := items[idx]
		results[idx] = &item
	}

	return results
}
