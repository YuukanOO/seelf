package builder_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

func Test_Builder(t *testing.T) {
	t.Run("should be able to express a basic select statement", func(t *testing.T) {
		q := builder.
			Query[any]("SELECT id, name FROM some_table WHERE name = ?", "john")

		assert.Equal(t, "SELECT id, name FROM some_table WHERE name = ?", q.String())
	})

	t.Run("should handle statements", func(t *testing.T) {
		var (
			id         = monad.Value(5)
			dummyFalse monad.Maybe[bool]
		)

		q := builder.
			Query[any]("SELECT id, name FROM some_table WHERE name = ?", "john").
			S(
				builder.MaybeValue(id, "AND id = ?"),
				builder.Maybe(dummyFalse, func(b bool) (string, []any) {
					return "AND name != ?", []any{"bob"}
				}),
				builder.Array("AND age IN", []int{18, 19}),
				builder.If(true, "AND TRUE"),
			).
			F("ORDER BY name")

		assert.Equal(t, "SELECT id, name FROM some_table WHERE name = ? AND id = ? AND age IN (?,?) AND TRUE ORDER BY name", q.String())
	})

	t.Run("should handle insert statements", func(t *testing.T) {
		q := builder.Insert("some_table", builder.Values{
			"name": "john",
			"age":  18,
			"id":   1,
		})

		assert.Match(t, "INSERT INTO some_table \\((,?(age|name|id)){3}\\) VALUES \\(\\?,\\?,\\?\\)", q.String())
	})

	t.Run("should handle update statements", func(t *testing.T) {
		q := builder.Update("some_table", builder.Values{
			"name": "bob",
			"age":  21,
		}).F("WHERE id = ?", 1)

		assert.Match(t, "UPDATE some_table SET (,?(age|name) = \\?){2} WHERE id = \\?", q.String())
	})
}
