package assert_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/numbers"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

func Test_True(t *testing.T) {
	t.Run("should correctly fail given a false value", func(t *testing.T) {
		mock := new(mockT)

		assert.True(mock, false, "with value %s", "false")

		shouldHaveFailed(t, mock, `should have been true - with value false
	expected:
true

	got:
false`)
	})

	t.Run("should correctly pass given a true value", func(t *testing.T) {
		mock := new(mockT)

		assert.True(mock, true, "with value %s", "true")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_False(t *testing.T) {
	t.Run("should correctly fail given a true value", func(t *testing.T) {
		mock := new(mockT)

		assert.False(mock, true, "with value %s", "true")

		shouldHaveFailed(t, mock, `should have been false - with value true
	expected:
false

	got:
true`)
	})

	t.Run("should correctly pass given a false value", func(t *testing.T) {
		mock := new(mockT)

		assert.False(mock, false, "with value %s", "false")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_Nil(t *testing.T) {
	t.Run("should correctly fail given a non nil value", func(t *testing.T) {
		mock := new(mockT)

		assert.Nil(mock, "a string", "with a non nil value")

		shouldHaveFailed(t, mock, `should have been nil - with a non nil value
	expected:
<nil>

	got:
"a string"`)
	})

	t.Run("should correctly pass given a nil value", func(t *testing.T) {
		mock := new(mockT)

		assert.Nil(mock, nil, "with a nil value")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_NotNil(t *testing.T) {
	t.Run("should correctly fail given a nil value", func(t *testing.T) {
		mock := new(mockT)

		assert.NotNil(mock, nil, "with a nil value")

		shouldHaveFailed(t, mock, `should have been not nil - with a nil value
	expected:
"nothing but <nil>"

	got:
<nil>`)
	})

	t.Run("should correctly pass given a non nil value", func(t *testing.T) {
		mock := new(mockT)

		assert.NotNil(mock, "a string", "with a non nil value")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_Equal(t *testing.T) {
	t.Run("should correctly fail given different values", func(t *testing.T) {
		mock := new(mockT)

		assert.Equal(mock, true, false, "with different values")

		shouldHaveFailed(t, mock, `should have been equal - with different values
	expected:
true

	got:
false`)
	})

	t.Run("should correctly pass given the expected value", func(t *testing.T) {
		mock := new(mockT)

		assert.Equal(mock, true, true, "with same values")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_NotEqual(t *testing.T) {
	t.Run("should correctly fail given the expected value", func(t *testing.T) {
		mock := new(mockT)

		assert.NotEqual(mock, true, true, "with same values")

		shouldHaveFailed(t, mock, `should not have been equal - with same values
	expected:
true

	got:
true`)
	})

	t.Run("should correctly pass given different values", func(t *testing.T) {
		mock := new(mockT)

		assert.NotEqual(mock, true, false, "with different values")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_DeepEqual(t *testing.T) {
	t.Run("should correctly fail given different slices", func(t *testing.T) {
		mock := new(mockT)

		assert.DeepEqual(mock, []int{1}, []int{2}, "with different slices")

		shouldHaveFailed(t, mock, `should have been deeply equal - with different slices
	expected:
[]int{1}

	got:
[]int{2}`)
	})

	t.Run("should correctly pass given the same slice", func(t *testing.T) {
		mock := new(mockT)

		assert.DeepEqual(mock, []int{1}, []int{1}, "with the same slice")

		shouldHaveSucceeded(t, mock)
	})

	t.Run("should correctly pass given the same struct", func(t *testing.T) {
		mock := new(mockT)

		assert.DeepEqual(mock, struct {
			foo string
			bar int
		}{foo: "bar", bar: 42}, struct {
			foo string
			bar int
		}{foo: "bar", bar: 42}, "with the same struct")

		shouldHaveSucceeded(t, mock)
	})

	t.Run("should correctly fail given different structs", func(t *testing.T) {
		mock := new(mockT)

		assert.DeepEqual(mock, struct {
			foo string
			bar int
		}{foo: "bar", bar: 42}, struct {
			foo string
			bar int
		}{foo: "bar", bar: 24}, "with different structs")

		shouldHaveFailed(t, mock, `should have been deeply equal - with different structs
	expected:
struct { foo string; bar int }{foo:"bar", bar:42}

	got:
struct { foo string; bar int }{foo:"bar", bar:24}`)
	})
}

func Test_Is(t *testing.T) {
	t.Run("should correctly fail given the wrong type", func(t *testing.T) {
		mock := new(mockT)

		result := assert.Is[string](mock, 5, "with wrong type")

		shouldHaveFailed(t, mock, `wrong type - with wrong type
	expected:
"string"

	got:
"int"`)

		if result != "" {
			t.Error("result should be empty")
		}
	})

	t.Run("should correctly pass given the right type", func(t *testing.T) {
		mock := new(mockT)

		result := assert.Is[string](mock, "test", "with right type")

		shouldHaveSucceeded(t, mock)

		if result != "test" {
			t.Error("result should be 'test'")
		}
	})
}

func Test_ErrorIs(t *testing.T) {
	t.Run("should correctly fail given a wrong error", func(t *testing.T) {
		mock := new(mockT)

		assert.ErrorIs(mock, errors.New("test"), errors.New("another err"), "with wrong error")

		shouldHaveFailed(t, mock, `errors should have match - with wrong error
	expected:
&errors.errorString{s:"test"}

	got:
&errors.errorString{s:"another err"}`)
	})

	t.Run("should correctly pass given a right error", func(t *testing.T) {
		mock := new(mockT)
		expectedErr := errors.New("test")
		actualErr := fmt.Errorf("with wrapped error %w", expectedErr)

		assert.ErrorIs(mock, expectedErr, actualErr, "with right error")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_HasLength(t *testing.T) {
	t.Run("should correctly fail given a wrong length", func(t *testing.T) {
		mock := new(mockT)

		assert.HasLength(mock, 5, []int{1, 2, 3}, "with wrong length")

		shouldHaveFailed(t, mock, `should have correct length - with wrong length
	expected:
5

	got:
3`)
	})

	t.Run("should correctly pass given a right length", func(t *testing.T) {
		mock := new(mockT)

		assert.HasLength(mock, 3, []int{1, 2, 3}, "with right length")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_HasNRunes(t *testing.T) {
	t.Run("should correctly fail given a wrong length", func(t *testing.T) {
		mock := new(mockT)

		assert.HasNRunes(mock, 5, "test", "with wrong length")

		shouldHaveFailed(t, mock, `should have correct number of characters - with wrong length
	expected:
5

	got:
4`)
	})

	t.Run("should correctly pass given a right length", func(t *testing.T) {
		mock := new(mockT)

		assert.HasNRunes(mock, 4, "test", "with right length")

		shouldHaveSucceeded(t, mock)
	})
}

type (
	eventA struct {
		bus.Notification
		value string
	}

	eventB struct {
		bus.Notification
		value int
	}

	entity struct {
		event.Emitter
	}
)

func (event eventA) Name_() string { return "eventA" }
func (event eventB) Name_() string { return "eventB" }

func Test_HasNEvents(t *testing.T) {
	ent := entity{}
	event.Store(&ent, eventA{}, eventB{})

	t.Run("should correctly fail given a wrong length", func(t *testing.T) {
		mock := new(mockT)

		assert.HasNEvents(mock, 1, &ent, "with wrong length")

		shouldHaveFailed(t, mock, `should have correct number of events - with wrong length
	expected:
1

	got:
2`)
	})

	t.Run("should correctly pass given a right length", func(t *testing.T) {
		mock := new(mockT)

		assert.HasNEvents(mock, 2, &ent, "with right length")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_EventIs(t *testing.T) {
	ent := entity{}
	a := eventA{value: "value"}
	b := eventB{value: 42}
	event.Store(&ent, a, b)

	t.Run("should fail if index is out of range", func(t *testing.T) {
		mock := new(mockT)

		result := assert.EventIs[eventA](mock, &ent, 2, "with wrong length")

		shouldHaveFailed(t, mock, `could not find an event at given index - with wrong length
	expected:
2

	got:
2`)

		if result == a {
			t.Error("result should be empty")
		}
	})

	t.Run("should fail if requested event type is wrong", func(t *testing.T) {
		mock := new(mockT)

		result := assert.EventIs[eventB](mock, &ent, 0, "with wrong event type")

		shouldHaveFailed(t, mock, `wrong type - with wrong event type
	expected:
"assert_test.eventB"

	got:
"assert_test.eventA"`)

		if result == b {
			t.Error("result should be empty")
		}
	})

	t.Run("should pass if requested event type is right", func(t *testing.T) {
		mock := new(mockT)

		result := assert.EventIs[eventA](mock, &ent, 0, "with right event type")

		shouldHaveSucceeded(t, mock)

		if result != a {
			t.Error("result should be equal to a")
		}
	})
}

func Test_ValidationError(t *testing.T) {
	t.Run("should fail if the error is not a validation one", func(t *testing.T) {
		mock := new(mockT)
		err := errors.New("test")

		assert.ValidationError(mock, validate.FieldErrors{}, err, "with wrong error type")

		shouldHaveFailed(t, mock, `wrong error type - with wrong error type
	expected:
"validate.FieldErrors"

	got:
"*errors.errorString"`)
	})

	t.Run("should fail if FieldErrors do not match", func(t *testing.T) {
		mock := new(mockT)
		err := validate.NewError(validate.FieldErrors{
			"a": numbers.ErrMin,
			"b": strings.ErrRequired,
		})

		assert.ValidationError(mock, validate.FieldErrors{
			"a": strings.ErrRequired,
			"b": numbers.ErrMin,
		}, err, "with wrong FieldErrors")

		shouldHaveFailed(t, mock, `should have been deeply equal - with wrong FieldErrors
	expected:
validate.FieldErrors{"a":apperr.Error{Code:"required", Detail:error(nil)}, "b":apperr.Error{Code:"min", Detail:error(nil)}}

	got:
validate.FieldErrors{"a":apperr.Error{Code:"min", Detail:error(nil)}, "b":apperr.Error{Code:"required", Detail:error(nil)}}`)
	})

	t.Run("should pass if FieldErrors match", func(t *testing.T) {
		mock := new(mockT)
		err := validate.NewError(validate.FieldErrors{
			"a": numbers.ErrMin,
			"b": strings.ErrRequired,
		})

		assert.ValidationError(mock, validate.FieldErrors{
			"a": numbers.ErrMin,
			"b": strings.ErrRequired,
		}, err, "with right FieldErrors")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_Zero(t *testing.T) {
	t.Run("should fail if the value is not the default one", func(t *testing.T) {
		mock := new(mockT)

		result := assert.Zero(mock, "test", "with a string")

		shouldHaveFailed(t, mock, `should be zero - with a string
	expected:
""

	got:
"test"`)

		if result != "test" {
			t.Error("result should be equal to the given value")
		}
	})

	t.Run("should pass if the value is the default one", func(t *testing.T) {
		mock := new(mockT)

		result := assert.Zero(mock, "", "with an empty string")

		shouldHaveSucceeded(t, mock)

		if result != "" {
			t.Error("result should be empty")
		}
	})
}

func Test_NotZero(t *testing.T) {
	t.Run("should fail if the value is the default one for simple types", func(t *testing.T) {
		mock := new(mockT)

		result := assert.NotZero(mock, "", "with an empty string")

		shouldHaveFailed(t, mock, `should not be zero - with an empty string
	expected:
"anything but the zero value"

	got:
""`)

		if result != "" {
			t.Error("result should be empty")
		}
	})

	t.Run("should pass if the value is not the default one for simple types", func(t *testing.T) {
		mock := new(mockT)

		result := assert.NotZero(mock, "test", "with a string")

		shouldHaveSucceeded(t, mock)

		if result != "test" {
			t.Error("result should be equal to the given value")
		}
	})

	t.Run("should fail if the value is the default one for complex types", func(t *testing.T) {
		mock := new(mockT)
		var time time.Time

		result := assert.NotZero(mock, time, "with a time.Time value")

		shouldHaveFailed(t, mock, `should not be zero - with a time.Time value
	expected:
"anything but the zero value"

	got:
time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)`)

		if result != time {
			t.Error("result should be empty")
		}
	})

	t.Run("should pass if the value is not the default one for complex types", func(t *testing.T) {
		mock := new(mockT)
		time := time.Now().UTC()

		result := assert.NotZero(mock, time, "with a time.Time value")

		shouldHaveSucceeded(t, mock)

		if result != time {
			t.Error("result should be equal to the given value")
		}
	})
}

func Test_Match(t *testing.T) {
	t.Run("should fail if a value does not match a regexp", func(t *testing.T) {
		mock := new(mockT)

		assert.Match(mock, "^test", "wrong value", "with a wrong string")

		shouldHaveFailed(t, mock, `should match - with a wrong string
	expected:
"^test"

	got:
"wrong value"`)
	})

	t.Run("should succeed if a value matches a regexp", func(t *testing.T) {
		mock := new(mockT)

		assert.Match(mock, "^test", "test", "with a valid string")

		shouldHaveSucceeded(t, mock)
	})
}

func Test_FileContentEquals(t *testing.T) {
	t.Run("should fail if the file does not exists", func(t *testing.T) {
		mock := new(mockT)

		assert.FileContentEquals(mock, "test", "not.txt", "with a non-existing file")

		shouldHaveFailed(t, mock, `should contains - with a non-existing file
	expected:
"test"

	got:
""`)
	})

	t.Run("should fail if the content does not match", func(t *testing.T) {
		mock := new(mockT)

		_ = os.WriteFile("wrong_content.txt", []byte("wrong content"), 0644)

		t.Cleanup(func() {
			_ = os.Remove("wrong_content.txt")
		})

		assert.FileContentEquals(mock, "test", "wrong_content.txt", "with a wrong content")

		shouldHaveFailed(t, mock, `should contains - with a wrong content
	expected:
"test"

	got:
"wrong content"`)
	})

	t.Run("should succeed if content matches", func(t *testing.T) {
		mock := new(mockT)

		_ = os.WriteFile("correct_content.txt", []byte("test"), 0644)

		t.Cleanup(func() {
			_ = os.Remove("correct_content.txt")
		})

		assert.FileContentEquals(mock, "test", "correct_content.txt", "with a correct content")

		shouldHaveSucceeded(t, mock)
	})
}

type mockT struct {
	testing.TB
	hasFailed bool
	msg       string
}

func (t *mockT) Errorf(format string, args ...any) {
	t.hasFailed = true
	t.msg = fmt.Sprintf(format, args...)
}

func shouldHaveFailed(t testing.TB, mock *mockT, expectedMessage string) {
	if !mock.hasFailed {
		t.Error("should have failed")
	}

	if mock.msg != expectedMessage {
		t.Errorf(`message should have matched:
expected:
	%s

got:
	%s`, expectedMessage, mock.msg)
	}
}

func shouldHaveSucceeded(t testing.TB, mock *mockT) {
	if mock.hasFailed {
		t.Error("should not have failed")
	}

	if mock.msg != "" {
		t.Error("message should be empty")
	}
}
