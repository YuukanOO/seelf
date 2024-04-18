package builder

import (
	"strings"

	"github.com/YuukanOO/seelf/pkg/monad"
)

// Maybe is a helper function that will only append the SQL statement and arguments
// if the monad value is set. This is useful for optional fields in a query.
func Maybe[T any](value monad.Maybe[T], fn func(T) (string, []any)) Statement {
	return func(builder Builder) {
		v, hasValue := value.TryGet()

		if !hasValue {
			return
		}

		sql, args := fn(v)

		builder.Apply(sql, args...)
	}
}

// Append the given sql string only if the expr is true.
func If(expr bool, sql string, args ...any) Statement {
	return func(builder Builder) {
		if !expr {
			return
		}

		builder.Apply(sql, args...)
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
	return func(builder Builder) {
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

		builder.Apply(prefix+" ("+placeholders+")", args...)
	}
}
