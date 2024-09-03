package strings_test

import (
	"regexp"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

func Test_Required(t *testing.T) {
	t.Run("should fail on empty or whitespaced strings", func(t *testing.T) {
		assert.ErrorIs(t, strings.ErrRequired, strings.Required(""))
		assert.ErrorIs(t, strings.ErrRequired, strings.Required("         "))
	})

	t.Run("should succeed on non-empty strings", func(t *testing.T) {
		assert.Nil(t, strings.Required("should be good"))
	})
}

func Test_Match(t *testing.T) {
	reUrlFormat := regexp.MustCompile("^https?://.+")

	t.Run("should fail on non matching strings", func(t *testing.T) {
		assert.ErrorIs(t, strings.ErrFormat, strings.Match(reUrlFormat)("some string"))
		assert.ErrorIs(t, strings.ErrFormat, strings.Match(reUrlFormat)("http://"))
	})

	t.Run("should succeed when matching", func(t *testing.T) {
		assert.Nil(t, strings.Match(reUrlFormat)("http://docker.localhost"))
	})
}

func Test_Min(t *testing.T) {
	t.Run("should fail on strings with less characters than the given length", func(t *testing.T) {
		assert.ErrorIs(t, strings.ErrMinLength, strings.Min(5)(""))
		assert.ErrorIs(t, strings.ErrMinLength, strings.Min(5)("test"))
	})

	t.Run("should succeed when enough characters are given", func(t *testing.T) {
		assert.Nil(t, strings.Min(5)("should be good"))
	})
}

func Test_Max(t *testing.T) {
	t.Run("should fail on strings with more characters than the given length", func(t *testing.T) {
		assert.ErrorIs(t, strings.ErrMaxLength, strings.Max(5)("should not be good"))
		assert.ErrorIs(t, strings.ErrMaxLength, strings.Max(5)("errorr"))
	})

	t.Run("should succeed when less characters than length are given", func(t *testing.T) {
		assert.Nil(t, strings.Max(5)("yeah!"))
	})
}
