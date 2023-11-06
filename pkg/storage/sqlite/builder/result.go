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
