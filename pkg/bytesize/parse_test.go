package bytesize_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bytesize"
)

func Test_Parse(t *testing.T) {
	t.Run("should be able to parse common format correctly", func(t *testing.T) {
		tests := []struct {
			value    string
			expected int64
			err      string
		}{
			{"1b", 1, ""},
			{"1kb", 1024, ""},
			{"56kb", 57344, ""},
			{"1mb", 1048576, ""},
			{"1.5mb", 1572864, ""},
			{"1 mb", 1048576, ""},
			{"1gb", 1073741824, ""},
			{"1bb", 0, "size: unrecognized suffix bb"},
			{"1", 0, "size: unrecognized suffix"},
			{"1..4mb", 0, "strconv.ParseFloat: parsing \"1..4\": invalid syntax"},
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				got, err := bytesize.Parse(tt.value)

				assert.Equal(t, tt.expected, got)

				if tt.err == "" {
					assert.Nil(t, err)
				} else {
					assert.Equal(t, tt.err, err.Error())
				}
			})
		}
	})
}
