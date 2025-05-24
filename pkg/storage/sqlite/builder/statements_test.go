package builder_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

func Test_OnConflictDoUpdate(t *testing.T) {
	t.Run("should correctly builds the statement", func(t *testing.T) {
		mock := &mockBuilder{}
		values := builder.Values{
			"key":       "1",
			"firstName": "John",
			"lastName":  "Doe",
		}

		builder.OnConflictDoUpdate("key", values, "firstName")(mock)

		assert.Equal(t, "ON CONFLICT (key) DO UPDATE SET lastName = ?", mock.query)
		assert.DeepEqual(t, []any{"Doe"}, mock.args)
	})
}

func Test_Maybe(t *testing.T) {
	t.Run("should apply the statement when value is set", func(t *testing.T) {
		mock := &mockBuilder{}
		value := monad.Value("test")

		builder.Maybe(value, func(v string) (string, []any) {
			return "WHERE name = ?", []any{v}
		})(mock)

		assert.Equal(t, "WHERE name = ?", mock.query)
		assert.DeepEqual(t, []any{"test"}, mock.args)
	})

	t.Run("should not apply the statement when value is not set", func(t *testing.T) {
		mock := &mockBuilder{}
		value := monad.None[string]()

		builder.Maybe(value, func(v string) (string, []any) {
			return "WHERE name = ?", []any{v}
		})(mock)

		assert.Equal(t, "", mock.query)
		assert.DeepEqual(t, []any(nil), mock.args)
	})
}

func Test_If(t *testing.T) {
	t.Run("should apply the statement when condition is true", func(t *testing.T) {
		mock := &mockBuilder{}

		builder.If(true, "WHERE active = ?", true)(mock)

		assert.Equal(t, "WHERE active = ?", mock.query)
		assert.DeepEqual(t, []any{true}, mock.args)
	})

	t.Run("should not apply the statement when condition is false", func(t *testing.T) {
		mock := &mockBuilder{}

		builder.If(false, "WHERE active = ?", true)(mock)

		assert.Equal(t, "", mock.query)
		assert.DeepEqual(t, []any(nil), mock.args)
	})
}

func Test_MaybeValue(t *testing.T) {
	t.Run("should apply the value when set", func(t *testing.T) {
		mock := &mockBuilder{}
		value := monad.Value(42)

		builder.MaybeValue(value, "WHERE id = ?")(mock)

		assert.Equal(t, "WHERE id = ?", mock.query)
		assert.DeepEqual(t, []any{42}, mock.args)
	})

	t.Run("should not apply the value when not set", func(t *testing.T) {
		mock := &mockBuilder{}
		value := monad.None[int]()

		builder.MaybeValue(value, "WHERE id = ?")(mock)

		assert.Equal(t, "", mock.query)
		assert.DeepEqual(t, []any(nil), mock.args)
	})
}

func Test_Array(t *testing.T) {
	t.Run("should generate IN clause with placeholders", func(t *testing.T) {
		mock := &mockBuilder{}
		values := []int{1, 2, 3}

		builder.Array("id IN", values)(mock)

		assert.Equal(t, "id IN (?,?,?)", mock.query)
		assert.DeepEqual(t, []any{1, 2, 3}, mock.args)
	})

	t.Run("should not apply anything when array is empty", func(t *testing.T) {
		mock := &mockBuilder{}
		values := []string{}

		builder.Array("name IN", values)(mock)

		assert.Equal(t, "", mock.query)
		assert.DeepEqual(t, []any(nil), mock.args)
	})
}

type mockBuilder struct {
	query string
	args  []any
}

func (m *mockBuilder) Apply(sql string, args ...any) {
	m.query = sql
	m.args = args
}
