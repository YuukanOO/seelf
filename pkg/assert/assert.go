package assert

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/validate"
)

// Asserts that the given value is true
func True[T ~bool](t testing.TB, actual T, formatAndMessage ...any) {
	if actual {
		return
	}

	failed(t, "should have been true", true, actual, formatAndMessage)
}

// Asserts that the given value is false
func False[T ~bool](t testing.TB, actual T, formatAndMessage ...any) {
	if !actual {
		return
	}

	failed(t, "should have been false", false, actual, formatAndMessage)
}

// Asserts that the given value is nil
func Nil(t testing.TB, actual any, formatAndMessage ...any) {
	if actual == nil {
		return
	}

	failed(t, "should have been nil", nil, actual, formatAndMessage)
}

// Asserts that the given value is not nil
func NotNil(t testing.TB, actual any, formatAndMessage ...any) {
	if actual != nil {
		return
	}

	failed(t, "should have been not nil", "nothing but <nil>", actual, formatAndMessage)
}

// Asserts that the given values are equal
func Equal[T comparable](t testing.TB, expected, actual T, formatAndMessage ...any) {
	if expected == actual {
		return
	}

	failed(t, "should have been equal", expected, actual, formatAndMessage)
}

// Assets that the given slices contains the same elements, not necessarily in the same order.
func ArrayEqual[T cmp.Ordered](t testing.TB, expected, actual []T, formatAndMessage ...any) {
	ArrayEqualFunc(t, expected, actual, cmp.Compare, formatAndMessage...)
}

// Same as ArrayEqual but for elements not implementing the cmp.Ordered.
func ArrayEqualFunc[T any](t testing.TB, expected, actual []T, equal func(T, T) int, formatAndMessage ...any) {
	if len(expected) != len(actual) {
		failed(t, "should have same length", len(expected), len(actual), formatAndMessage)
		return
	}

	slices.SortFunc(expected, equal)
	slices.SortFunc(actual, equal)

	if reflect.DeepEqual(expected, actual) {
		return
	}

	failed(t, "should have been equal", expected, actual, formatAndMessage)
}

// Asserts that the given values are not equal
func NotEqual[T comparable](t testing.TB, expected, actual T, formatAndMessage ...any) {
	if expected != actual {
		return
	}

	failed(t, "should not have been equal", expected, actual, formatAndMessage)
}

// Asserts that the given values are deeply equal using the reflect.DeepEqual function
func DeepEqual[T any](t testing.TB, expected, actual T, formatAndMessage ...any) {
	if reflect.DeepEqual(expected, actual) {
		return
	}

	failed(t, "should have been deeply equal", expected, actual, formatAndMessage)
}

// Asserts that the given value is of the given type and returns it.
func Is[T any](t testing.TB, actual any, formatAndMessage ...any) T {
	result, ok := actual.(T)

	if ok {
		return result
	}

	failed(t, "wrong type", reflect.TypeOf(result).String(), reflect.TypeOf(actual).String(), formatAndMessage)

	return result
}

// Asserts that the given error is the expected error using the function errors.Is
func ErrorIs(t testing.TB, expected, actual error, formatAndMessage ...any) {
	if errors.Is(actual, expected) {
		return
	}

	failed(t, "errors should have match", expected, actual, formatAndMessage)
}

// Asserts that the actual slice has the expected length
func HasLength[T any](t testing.TB, expected int, actual []T, formatAndMessage ...any) {
	got := len(actual)

	if got == expected {
		return
	}

	failed(t, "should have correct length", expected, got, formatAndMessage)
}

// Asserts that the actual string has the expected number of utf8 runes
func HasNRunes[T ~string](t testing.TB, expected int, actual T, formatAndMessage ...any) {
	got := utf8.RuneCountInString(string(actual))

	if got == expected {
		return
	}

	failed(t, "should have correct number of characters", expected, got, formatAndMessage)
}

// Asserts that the actual source has the expected number of events
func HasNEvents[T event.Source](t testing.TB, expected int, source T, formatAndMessage ...any) {
	_, events := event.Unwrap(source)
	got := len(events)

	if got == expected {
		return
	}

	failed(t, "should have correct number of events", expected, got, formatAndMessage)
}

// Asserts that the actual source has the expected event type at the given index and returns it
func EventIs[T event.Event](t testing.TB, source event.Source, index int, formatAndMessage ...any) T {
	_, events := event.Unwrap(source)

	if index >= len(events) {
		failed(t, "could not find an event at given index", index, len(events), formatAndMessage)
		var r T
		return r
	}

	return Is[T](t, events[index], formatAndMessage...)
}

// Asserts that the actual error is a validation error with the expected field errors
func ValidationError(t testing.TB, expected validate.FieldErrors, actual error, formatAndMessage ...any) {
	ErrorIs(t, validate.ErrValidationFailed, actual, formatAndMessage...)

	fields, ok := apperr.As[validate.FieldErrors](actual)

	if !ok {
		failed(t, "wrong error type", reflect.TypeOf(expected).String(), reflect.TypeOf(actual).String(), formatAndMessage)
		return
	}

	DeepEqual(t, expected, fields, formatAndMessage...)
}

// Asserts that the given value is the zero value for the corresponding type
func Zero[T comparable](t testing.TB, actual T, formatAndMessage ...any) T {
	var zero T

	if actual == zero {
		return actual
	}

	failed(t, "should be zero", zero, actual, formatAndMessage)

	return actual
}

// Asserts that the given value is not the zero value for the corresponding type and returns it
func NotZero[T comparable](t testing.TB, actual T, formatAndMessage ...any) T {
	var zero T

	if actual != zero {
		return actual
	}

	failed(t, "should not be zero", "anything but the zero value", actual, formatAndMessage)

	return actual
}

// Asserts that the given value matches the expected regular expression
func Match(t testing.TB, expectedRegexp string, value string, formatAndMessage ...any) {
	if regexp.MustCompile(expectedRegexp).MatchString(value) {
		return
	}

	failed(t, "should match", expectedRegexp, value, formatAndMessage)
}

// Asserts that the file at the given path contains the expected content
func FileContentEquals(t testing.TB, expectedContent string, path string, formatAndMessage ...any) {
	data, _ := os.ReadFile(path)
	str := string(data)

	if str == expectedContent {
		return
	}

	failed(t, "should contains", expectedContent, str, formatAndMessage)
}

func failed(t testing.TB, msg string, expected, actual any, contextMessage []any) {
	if len(contextMessage) > 0 {
		msg = fmt.Sprintf("%s - %s", msg, fmt.Sprintf(contextMessage[0].(string), contextMessage[1:]...))
	}

	t.Errorf(`%s
	expected:
%#v

	got:
%#v`, msg, expected, actual)
}
