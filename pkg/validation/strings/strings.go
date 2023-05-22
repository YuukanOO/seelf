package strings

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/validation"
)

var (
	ErrRequired  = apperr.New("required")
	ErrMinLength = apperr.New("min_length")
	ErrMaxLength = apperr.New("max_length")
	ErrFormat    = apperr.New("invalid_format")
)

func Required(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrRequired
	}

	return nil
}

func Match(expr *regexp.Regexp) validation.Validator[string] {
	return func(value string) error {
		if !expr.MatchString(value) {
			return ErrFormat
		}

		return nil
	}
}

func Min(length int) validation.Validator[string] {
	return func(value string) error {
		if utf8.RuneCountInString(value) < length {
			return ErrMinLength
		}

		return nil
	}
}

func Max(length int) validation.Validator[string] {
	return func(value string) error {
		if utf8.RuneCountInString(value) > length {
			return ErrMaxLength
		}

		return nil
	}
}
