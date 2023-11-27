package monad_test

import (
	"encoding/json"
	"testing"

	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Patch(t *testing.T) {
	t.Run("should default to a not set, empty value", func(t *testing.T) {
		var p monad.Patch[int]

		testutil.IsFalse(t, p.IsSet())
		testutil.IsFalse(t, p.IsNil())
		testutil.IsFalse(t, p.HasValue())
	})

	t.Run("should be instantiable with a value", func(t *testing.T) {
		p := monad.PatchValue(42)

		testutil.IsTrue(t, p.IsSet())
		testutil.IsFalse(t, p.IsNil())
		testutil.IsTrue(t, p.HasValue())
		testutil.Equals(t, 42, p.MustGet())
	})

	t.Run("should be instantiable with a nil value", func(t *testing.T) {
		p := monad.Nil[int]()

		testutil.IsTrue(t, p.IsSet())
		testutil.IsTrue(t, p.IsNil())
		testutil.IsFalse(t, p.HasValue())
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

				testutil.Equals(t, test.isSet, isSet)
				testutil.Equals(t, test.hasValue, m.HasValue())
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

				testutil.IsNil(t, json.Unmarshal([]byte(test.json), &value))
				testutil.Equals(t, test.isSet, value.Number.IsSet())
				testutil.Equals(t, test.isNil, value.Number.IsNil())
				testutil.Equals(t, test.hasValue, value.Number.HasValue())
			})
		}
	})
}

type someStruct struct {
	Number monad.Patch[int] `json:"number"`
}
