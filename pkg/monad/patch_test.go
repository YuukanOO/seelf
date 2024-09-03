package monad_test

import (
	"encoding/json"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
)

func Test_Patch(t *testing.T) {
	t.Run("should default to a not set, empty value", func(t *testing.T) {
		var p monad.Patch[int]

		assert.False(t, p.IsSet())
		assert.False(t, p.IsNil())
		assert.False(t, p.HasValue())
	})

	t.Run("should be instantiable with a value", func(t *testing.T) {
		p := monad.PatchValue(42)

		assert.True(t, p.IsSet())
		assert.False(t, p.IsNil())
		assert.True(t, p.HasValue())
		assert.Equal(t, 42, p.MustGet())
	})

	t.Run("should be instantiable with a nil value", func(t *testing.T) {
		p := monad.Nil[int]()

		assert.True(t, p.IsSet())
		assert.True(t, p.IsNil())
		assert.False(t, p.HasValue())
	})

	t.Run("should return the inner monad and a boolean indicating if it has been set", func(t *testing.T) {
		tests := []struct {
			name     string
			value    monad.Patch[int]
			isSet    bool
			hasValue bool
		}{
			{"empty patch", monad.Patch[int]{}, false, false},
			{"nil patch", monad.Nil[int](), true, false},
			{"patch with a value", monad.PatchValue(42), true, true},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				m, isSet := test.value.TryGet()

				assert.Equal(t, test.isSet, isSet)
				assert.Equal(t, test.hasValue, m.HasValue())
			})
		}
	})

	t.Run("should correctly handle a JSON when unmarshalling", func(t *testing.T) {
		tests := []struct {
			json     string
			isSet    bool
			isNil    bool
			hasValue bool
		}{
			{"{}", false, false, false},
			{`{ "number": null }`, true, true, false},
			{`{ "number": 42 }`, true, false, true},
		}

		for _, test := range tests {
			t.Run(test.json, func(t *testing.T) {
				var value someStruct

				assert.Nil(t, json.Unmarshal([]byte(test.json), &value))
				assert.Equal(t, test.isSet, value.Number.IsSet())
				assert.Equal(t, test.isNil, value.Number.IsNil())
				assert.Equal(t, test.hasValue, value.Number.HasValue())
			})
		}
	})
}

type someStruct struct {
	Number monad.Patch[int] `json:"number"`
}
