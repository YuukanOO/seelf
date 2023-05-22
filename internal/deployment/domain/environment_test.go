package domain_test

import (
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Environment(t *testing.T) {
	t.Run("should validates input string", func(t *testing.T) {
		tests := []struct {
			input string
			valid bool
		}{
			{"some input", false},
			{"productionnn", false},
			{"production", true},
			{"staging", true},
		}

		for _, test := range tests {
			t.Run(test.input, func(t *testing.T) {
				r, err := domain.EnvironmentFrom(test.input)

				if test.valid {
					testutil.Equals(t, domain.Environment(test.input), r)
					testutil.IsNil(t, err)
				} else {
					testutil.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
					testutil.Equals(t, "", r)
				}
			})
		}
	})

	t.Run("should expose a method to check if an env is the production one", func(t *testing.T) {
		tests := []struct {
			input      domain.Environment
			production bool
		}{
			{"staging", false},
			{"production", true},
		}

		for _, test := range tests {
			t.Run(string(test.input), func(t *testing.T) {
				testutil.Equals(t, test.production, test.input.IsProduction())
			})
		}
	})
}

func Test_EnvironmentsEnv(t *testing.T) {
	t.Run("should require valid environment names", func(t *testing.T) {
		rawEnvs := map[string]map[string]map[string]string{
			"production":      {"app": {}},
			"not a valid env": {"app": {}},
		}

		r, err := domain.EnvironmentsEnvFrom(rawEnvs)

		testutil.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
		testutil.DeepEquals(t, domain.EnvironmentsEnv{}, r)
	})

	t.Run("should builds a map from a raw one", func(t *testing.T) {
		rawEnvs := map[string]map[string]map[string]string{
			"production": {"app": {"DEBUG": "false"}},
			"staging":    {"app": {"DEBUG": "true"}},
		}

		r, err := domain.EnvironmentsEnvFrom(rawEnvs)

		testutil.IsNil(t, err)
		testutil.DeepEquals(t, domain.EnvironmentsEnv{
			"production": {"app": {"DEBUG": "false"}},
			"staging":    {"app": {"DEBUG": "true"}},
		}, r)
	})

	t.Run("should be able to compare itself with another envs map", func(t *testing.T) {
		tests := []struct {
			a        domain.EnvironmentsEnv
			b        domain.EnvironmentsEnv
			expected bool
		}{
			{nil, nil, true},
			{
				a:        nil,
				b:        domain.EnvironmentsEnv{},
				expected: false,
			},
			{
				a:        domain.EnvironmentsEnv{"production": {}},
				b:        domain.EnvironmentsEnv{"production": {}},
				expected: true,
			},
			{
				a:        domain.EnvironmentsEnv{"production": {"another": {"level": "hey"}}},
				b:        domain.EnvironmentsEnv{"production": {}},
				expected: false,
			},
			{
				a:        domain.EnvironmentsEnv{"production": {"another": {"level": "hey"}}},
				b:        domain.EnvironmentsEnv{"production": {"another": {"level": "hey"}}},
				expected: true,
			},
			{
				a:        domain.EnvironmentsEnv{"production": {"another": {"level": "hey"}}},
				b:        domain.EnvironmentsEnv{"production": {"another": {"level": "nope"}}},
				expected: false,
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%v %v", test.a, test.b), func(t *testing.T) {
				r := test.a.Equals(test.b)
				testutil.Equals(t, test.expected, r)

				r = test.b.Equals(test.a)
				testutil.Equals(t, test.expected, r)
			})
		}
	})
}
