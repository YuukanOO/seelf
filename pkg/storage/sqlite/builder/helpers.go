package builder

import "github.com/YuukanOO/seelf/pkg/storage"

// Map TChild entities and merge them in a TParent entity using the key/index mapping
// given by the KeyedResult.
//
// The mapper given to this function SHOULD NOT scan the first column of the result set
// because this function will do it and consider the first scanned value to be the key
// of the TParent entity. It means that you should always include the TParent key
// as the first column when retrieving related entities.
func Merge[TParent, TChildren any](
	results KeyedResult[TParent],
	mapper storage.Mapper[TChildren],
	merger storage.Merger[TParent, TChildren],
) storage.Mapper[TChildren] {
	return func(s storage.Scanner) (data TChildren, err error) {
		var key string
		data, err = mapper(keyScanner(s, &key))

		idx, found := results.indexByKeys[key]

		if found {
			results.data[idx] = merger(results.data[idx], data)
		}

		return data, err
	}
}

// Tiny mapper when all you have to do is retrieve a single value from a query.
func extract[T any](scanner storage.Scanner) (value T, err error) {
	err = scanner.Scan(&value)
	return value, err
}

type keyScannerDecorator struct {
	target    *string
	decorated storage.Scanner
}

// Wrap the given scanner to extract the key from the first column returned by the
// scanner. Used by the Merge function to retrieve the key of the parent entity and
// merge the result with the child entity.
func keyScanner(s storage.Scanner, target *string) storage.Scanner {
	return &keyScannerDecorator{target, s}
}

func (s *keyScannerDecorator) Scan(dest ...any) error {
	dest = append([]any{s.target}, dest...)
	return s.decorated.Scan(dest...)
}
