package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_AppNameFrom(t *testing.T) {
	t.Run("should validates input string", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"", false},
			{"  some-app", false},
			{"some-app   ", false},
			{"some-app-with-ç-special-char", false},
			{"WITH-caps", false},
			{"My app", false},
			{"some-app-1337", true},
			{"my-app", true},
			{"some-app-stagin", true},
			{"some-app-staging", false},
		}

		for _, test := range tests {
			t.Run(test.input, func(t *testing.T) {
				r, err := domain.AppNameFrom(test.input)

				if test.valid {
					assert.Nil(t, err)
					assert.Equal(t, domain.AppName(test.input), r)
				} else {
					assert.ErrorIs(t, domain.ErrInvalidAppName, err)
					assert.Equal(t, "", r)
				}
			})
		}
	})
}
