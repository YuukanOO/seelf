package strings_test

import (
	"regexp"
	"testing"

	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

func Test_Required(t *testing.T) {
	t.Run("should fail on empty or whitespaced strings", func(t *testing.T) {
		testutil.ErrorIs(t, strings.ErrRequired, strings.Required(""))
		testutil.ErrorIs(t, strings.ErrRequired, strings.Required("         "))
	})

	t.Run("should succeed on non-empty strings", func(t *testing.T) {
		testutil.IsNil(t, strings.Required("should be good"))
	})
}

func Test_Match(t *testing.T) {
	reUrlFormat := regexp.MustCompile("^https?://.+")

	t.Run("should fail on non matching strings", func(t *testing.T) {
		testutil.ErrorIs(t, strings.ErrFormat, strings.Match(reUrlFormat)("some string"))
		testutil.ErrorIs(t, strings.ErrFormat, strings.Match(reUrlFormat)("http://"))
	})

	t.Run("should succeed when matching", func(t *testing.T) {
		testutil.IsNil(t, strings.Match(reUrlFormat)("http://docker.localhost"))
	})
}

func Test_Min(t *testing.T) {
	t.Run("should fail on strings with less characters than the given length", func(t *testing.T) {
		testutil.ErrorIs(t, strings.ErrMinLength, strings.Min(5)(""))
		testutil.ErrorIs(t, strings.ErrMinLength, strings.Min(5)("test"))
	})

	t.Run("should succeed when enough characters are given", func(t *testing.T) {
		testutil.IsNil(t, strings.Min(5)("should be good"))
	})
}

func Test_Max(t *testing.T) {
	t.Run("should fail on strings with more characters than the given length", func(t *testing.T) {
		testutil.ErrorIs(t, strings.ErrMaxLength, strings.Max(5)("should not be good"))
		testutil.ErrorIs(t, strings.ErrMaxLength, strings.Max(5)("errorr"))
	})

	t.Run("should succeed when less characters than length are given", func(t *testing.T) {
		testutil.IsNil(t, strings.Max(5)("yeah!"))
	})
}
