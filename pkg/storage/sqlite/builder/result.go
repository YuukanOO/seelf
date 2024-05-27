package builder

import "golang.org/x/exp/maps"

type (
	keysMapping map[string]int

	// Represents a key indexed set of data.
	keyedResult[T any] struct {
		data        []T
		indexByKeys keysMapping
	}
)

// Keys returns the list of keys contained in this dataset.
func (r *keyedResult[T]) Keys() []string { return maps.Keys(r.indexByKeys) }

// Update the result with the given key by applying the given function if it exists.
func (r *keyedResult[T]) Update(targetKey string, updateFn func(T) T) {
	idx, found := r.indexByKeys[targetKey]

	if !found {
		return
	}

	r.data[idx] = updateFn(r.data[idx])
}
