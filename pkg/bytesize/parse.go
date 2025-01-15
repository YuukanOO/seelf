package bytesize

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

const (
	b  float64 = 1
	kb float64 = 1 << (10 * iota)
	mb
	gb
)

var (
	unitMap = map[string]float64{
		"b":  b,
		"kb": kb,
		"mb": mb,
		"gb": gb,
	}

	errUnrecognizedSuffix = errors.New("size: unrecognized suffix")
)

// Parse the given string representing a size in the binary format but using common international
// suffixes such as b, kb, mb, and gb.
//
// Returns the byte size as an int64 because that's what the rest of Size related methods use.
//
// Inspired by https://github.com/inhies/go-bytesize/
func Parse(value string) (int64, error) {
	// Remove leading and trailing whitespace
	value = strings.TrimSpace(value)

	split := make([]string, 0)
	for i, r := range value {
		if !unicode.IsDigit(r) && r != '.' {
			// Split the string by digit and size designator, remove whitespace
			split = append(split, strings.TrimSpace(value[:i]))
			split = append(split, strings.TrimSpace(value[i:]))
			break
		}
	}

	if len(split) != 2 {
		return 0, errUnrecognizedSuffix
	}

	unit, ok := unitMap[strings.ToLower(split[1])]
	if !ok {
		return 0, errors.New("size: unrecognized suffix " + split[1])

	}

	fv, err := strconv.ParseFloat(split[0], 64)

	if err != nil {
		return 0, err
	}

	return int64(fv * unit), nil
}
