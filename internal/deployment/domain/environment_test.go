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

func Test_ServicesEnv(t *testing.T) {
	t.Run("should builds a map from a raw one", func(t *testing.T) {
		rawEnvs := map[string]map[string]string{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "admin"},
		}

		r := domain.ServicesEnvFrom(rawEnvs)

		testutil.DeepEquals(t, domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "admin"},
		}, r)
	})

	t.Run("should returns an empty map if the raw one is nil", func(t *testing.T) {
		r := domain.ServicesEnvFrom(nil)

		testutil.DeepEquals(t, domain.ServicesEnv{}, r)
	})

	t.Run("should skip nil environment variables values", func(t *testing.T) {
		rawEnvs := map[string]map[string]string{
			"app": {"DEBUG": "false"},
			"db":  nil,
		}

		r := domain.ServicesEnvFrom(rawEnvs)

		testutil.DeepEquals(t, domain.ServicesEnv{
			"app": {"DEBUG": "false"},
		}, r)
	})

	t.Run("should implement the Valuer interface", func(t *testing.T) {
		str, err := domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "admin"},
		}.Value()

		testutil.IsNil(t, err)

		testutil.Equals(t, `{"app":{"DEBUG":"false"},"db":{"USERNAME":"admin"}}`, str)
	})

	t.Run("should implement the Scanner interface", func(t *testing.T) {
		var r domain.ServicesEnv

		err := r.Scan(`{"app":{"DEBUG":"false"},"db":{"USERNAME":"admin"}}`)

		testutil.IsNil(t, err)
		testutil.DeepEquals(t, domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "admin"},
		}, r)
	})
}

func Test_EnvironmentConfig(t *testing.T) {
	t.Run("should be able to build a new environment config", func(t *testing.T) {
		target := domain.TargetID("target")

		r := domain.NewEnvironmentConfig(target)

		testutil.Equals(t, target, r.Target())
		testutil.IsFalse(t, r.Vars().HasValue())
	})

	t.Run("should be able to configure environment variables", func(t *testing.T) {
		target := domain.TargetID("target")
		vars := domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "admin"},
		}

		r := domain.NewEnvironmentConfig(target)
		r.HasEnvironmentVariables(vars)

		testutil.Equals(t, target, r.Target())
		testutil.IsTrue(t, r.Vars().HasValue())
		testutil.DeepEquals(t, vars, r.Vars().MustGet())
	})

	t.Run("should be able to compare itself with another config", func(t *testing.T) {
		tests := []struct {
			a        func() domain.EnvironmentConfig
			b        func() domain.EnvironmentConfig
			expected bool
		}{
			{
				a:        func() domain.EnvironmentConfig { return domain.NewEnvironmentConfig("1") },
				b:        func() domain.EnvironmentConfig { return domain.NewEnvironmentConfig("1") },
				expected: true,
			},
			{
				a:        func() domain.EnvironmentConfig { return domain.NewEnvironmentConfig("1") },
				b:        func() domain.EnvironmentConfig { return domain.NewEnvironmentConfig("2") },
				expected: false,
			},
			{
				a: func() domain.EnvironmentConfig {
					conf := domain.NewEnvironmentConfig("1")
					conf.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "false"}})
					return conf
				},
				b: func() domain.EnvironmentConfig {
					conf := domain.NewEnvironmentConfig("1")
					conf.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "false"}})
					return conf
				},
				expected: true,
			},
			{
				a: func() domain.EnvironmentConfig {
					conf := domain.NewEnvironmentConfig("1")
					conf.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "false"}})
					return conf
				},
				b:        func() domain.EnvironmentConfig { return domain.NewEnvironmentConfig("1") },
				expected: false,
			},
			{
				a: func() domain.EnvironmentConfig {
					conf := domain.NewEnvironmentConfig("1")
					conf.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "false"}})
					return conf
				},
				b: func() domain.EnvironmentConfig {
					conf := domain.NewEnvironmentConfig("1")
					conf.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "true"}})
					return conf
				},
				expected: false,
			},
		}

		for _, test := range tests {
			a := test.a()
			b := test.b()
			t.Run(fmt.Sprintf("%v %v", a, b), func(t *testing.T) {
				r := a.Equals(b)
				testutil.Equals(t, test.expected, r)

				r = b.Equals(a)
				testutil.Equals(t, test.expected, r)
			})
		}
	})
}
