package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Config(t *testing.T) {
	app := domain.NewApp("my-app", "uid")

	t.Run("should be created with ConfigFrom", func(t *testing.T) {
		conf := domain.NewConfig(app, domain.Production)

		testutil.Equals(t, "my-app", conf.AppName())
		testutil.Equals(t, domain.Production, conf.Environment())
		testutil.IsFalse(t, conf.Env().HasValue())
		testutil.IsFalse(t, conf.EnvironmentVariablesFor("app").HasValue())
	})

	t.Run("should retrieve env if available from an app", func(t *testing.T) {
		e := domain.ServicesEnv{
			"app": {},
		}
		a := domain.NewApp("my-app", "uid")
		a.HasEnvironmentVariables(domain.EnvironmentsEnv{
			"production": e,
		})

		conf := domain.NewConfig(a, domain.Production)

		testutil.IsTrue(t, conf.Env().HasValue())
		testutil.DeepEquals(t, e, conf.Env().MustGet())
	})

	t.Run("should provide a way to retrieve environment variables for a service name", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		a.HasEnvironmentVariables(domain.EnvironmentsEnv{
			"integration": {
				"app": {
					"DEBUG": "true",
				},
			},
		})

		conf := domain.NewConfig(a, "integration")

		testutil.IsFalse(t, conf.EnvironmentVariablesFor("db").HasValue())
		testutil.IsTrue(t, conf.EnvironmentVariablesFor("app").HasValue())
		testutil.DeepEquals(t, domain.EnvVars{
			"DEBUG": "true",
		}, conf.EnvironmentVariablesFor("app").MustGet())
	})

	t.Run("should have a subdomain equals to app name if env is production", func(t *testing.T) {
		conf := domain.NewConfig(app, domain.Production)
		testutil.Equals(t, "my-app", conf.SubDomain())
	})

	t.Run("should have a subdomain suffixed by the env if not production", func(t *testing.T) {
		conf := domain.NewConfig(app, "integration")
		testutil.Equals(t, "my-app-integration", conf.SubDomain())
	})

	t.Run("should expose a project name", func(t *testing.T) {
		conf := domain.NewConfig(app, "integration")
		testutil.Equals(t, "my-app-integration", conf.ProjectName())
	})
}
