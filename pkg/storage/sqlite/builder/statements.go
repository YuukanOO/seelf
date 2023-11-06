package builder

import (
	"fmt"
	"strings"

	"github.com/YuukanOO/seelf/pkg/monad"
)

// Maybe is a helper function that will only append the SQL statement and arguments
// if the monad value is set. This is useful for optional fields in a query.
func Maybe[T any](value monad.Maybe[T], fn func(T) (string, []any)) Statement {
	return func(builder sqlBuilder) {
		if !value.HasValue() {
			return
		}

		sql, args := fn(value.MustGet())

		builder.apply(sql, args...)
	}
}

// Shortcut to append the monad value to the query if it's set.
func MaybeValue[T any](value monad.Maybe[T], sql string) Statement {
	return Maybe(value, func(v T) (string, []any) {
		return sql, []any{v}
	})
}

// Generates an IN clause from a list of values. Just provide the prefix such as
// "name IN" and the list of values and it will append (?, ?, ?) to the prefix based
// on what's in the list.
func Array[T any](prefix string, values []T) Statement {
	return func(builder sqlBuilder) {
		size := len(values)

		if size == 0 {
			return
		}

		var (
			placeholders = strings.Repeat(",?", size)[1:] // Remove the first comma
			args         = make([]any, size)
		)

		for i, value := range values {
			args[i] = value
		}

		builder.apply(fmt.Sprintf("%s (%s)", prefix, placeholders), args...)
	}
}
