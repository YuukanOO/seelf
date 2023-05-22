// Package testutil exposes assert utilities used in the project to make things
// simpler to read.
package testutil

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/YuukanOO/seelf/pkg/event"
)

func Equals[T comparable](t testing.TB, expected, actual T) {
	if expected != actual {
		expectationVersusReality(t, "should have been equals", expected, actual)
	}
}

func NotEquals[T comparable](t testing.TB, expected, actual T) {
	if expected == actual {
		expectationVersusReality(t, "should not have been equals", expected, actual)
	}
}

func DeepEquals[T any](t testing.TB, expected, actual T) {
	if !reflect.DeepEqual(expected, actual) {
		expectationVersusReality(t, "should have been deeply equals", expected, actual)
	}
}

func IsTrue(t testing.TB, expr bool) {
	Equals(t, true, expr)
}

func IsFalse(t testing.TB, expr bool) {
	Equals(t, false, expr)
}

func IsNil(t testing.TB, expr any) {
	if expr != nil {
		expectationVersusReality(t, "should have been nil", nil, expr)
	}
}

func IsNotNil(t testing.TB, expr any) {
	if expr == nil {
		expectationVersusReality(t, "should have been not nil", "nothing but <nil>", expr)
	}
}

func HasLength[T any](t testing.TB, arr []T, length int) {
	actual := len(arr)
	if actual != length {
		expectationVersusReality(t, "should have correct size", length, actual)
	}
}

func HasNChars[T ~string](t testing.TB, expected int, value T) {
	actual := utf8.RuneCountInString(string(value))

	if actual != expected {
		expectationVersusReality(t, "should have correct number of characters", expected, actual)
	}
}

func Contains(t testing.TB, expected string, value string) {
	if !strings.Contains(value, expected) {
		expectationVersusReality(t, "should contains the string", expected, value)
	}
}

func Match(t testing.TB, re string, value string) {
	if !regexp.MustCompile(re).MatchString(value) {
		expectationVersusReality(t, "should match", re, value)
	}
}

func ErrorIs(t testing.TB, expected, actual error) {
	if !errors.Is(actual, expected) {
		expectationVersusReality(t, "errors should have match", expected, actual)
	}
}

func HasNEvents(t testing.TB, source event.Source, expected int) {
	actual := len(event.Unwrap(source))

	if actual != expected {
		expectationVersusReality(t, "should have correct number of events", expected, actual)
	}
}

func EventIs[T event.Event](t testing.TB, source event.Source, index int) (result T) {
	events := event.Unwrap(source)

	if index >= len(events) {
		expectationVersusReality(t, "could not find an event at given index", index, nil)
		return result
	}

	result, ok := events[index].(T)

	if !ok {
		expectationVersusReality(t, "wrong event type", events[index], result)
		return result
	}

	return result
}

func expectationVersusReality(t testing.TB, message string, expected, actual any) {
	t.Fatalf(`%s
	expected:
%v

	got:
%v`, message, expected, actual)
}
