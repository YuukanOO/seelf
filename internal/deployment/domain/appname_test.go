package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
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
			{"My app", false},
			{"some-app-1337", true},
			{"my-app", true},
		}

		for _, test := range tests {
			t.Run("", func(t *testing.T) {
				r, err := domain.AppNameFrom(test.input)

				if test.valid {
					testutil.Equals(t, domain.AppName(test.input), r)
					testutil.IsNil(t, err)
				} else {
					testutil.Equals(t, "", r)
					testutil.ErrorIs(t, domain.ErrInvalidAppName, err)
				}
			})
		}
	})
}
