package builder

import "golang.org/x/exp/maps"

type (
	KeysMapping map[string]int

	// Represents a key indexed set of data.
	KeyedResult[T any] struct {
		data        []T
		indexByKeys KeysMapping
	}
)

// Keys returns the list of keys contained in this dataset.
func (r KeyedResult[T]) Keys() []string { return maps.Keys(r.indexByKeys) }

// Update the result with the given key by applying the given function if it exists.
func (r *KeyedResult[T]) Update(targetKey string, updateFn func(T) T) {
	idx, found := r.indexByKeys[targetKey]

	if !found {
		return
	}

	r.data[idx] = updateFn(r.data[idx])
}
