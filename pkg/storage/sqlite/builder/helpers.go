package builder

import "github.com/YuukanOO/seelf/pkg/storage"

// Merge the results of the query with given data. The mapper MUST returns the key
// representing the parent entity identity.
//
// With the retrieved key, it will merge the result by calling the merger func.
func Merge[TParent, TChildren any](
	results KeyedResult[TParent],
	mapper storage.KeyedMapper[TChildren, storage.Scanner],
	merger storage.Merger[TParent, TChildren],
) storage.Mapper[TChildren] {
	return func(s storage.Scanner) (data TChildren, err error) {
		key, data, err := mapper(s)

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
