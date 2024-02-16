package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Config(t *testing.T) {
	production := domain.
		NewEnvironmentConfig("production-target").
		WithEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "prodadmin"},
		})

	staging := domain.NewEnvironmentConfig("staging-target")
	app := must.Panic(domain.NewApp("my-app", production, staging, domain.AppNamingAvailable, "uid"))

	t.Run("could be created from an app", func(t *testing.T) {
		conf, err := app.ConfigSnapshotFor(domain.Production)

		testutil.IsNil(t, err)
		testutil.Equals(t, "my-app", conf.AppName())
		testutil.Equals(t, domain.Production, conf.Environment())
		testutil.Equals(t, production.Target(), conf.Target())
		testutil.DeepEquals(t, production.Vars(), conf.Vars())
	})

	t.Run("should fail if env is not valid", func(t *testing.T) {
		_, err := app.ConfigSnapshotFor("invalid")

		testutil.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
	})

	t.Run("should provide a way to retrieve environment variables for a service name", func(t *testing.T) {
		conf, _ := app.ConfigSnapshotFor(domain.Production)

		testutil.IsFalse(t, conf.EnvironmentVariablesFor("otherservice").HasValue())
		testutil.IsTrue(t, conf.EnvironmentVariablesFor("app").HasValue())
		testutil.DeepEquals(t, domain.EnvVars{
			"DEBUG": "false",
		}, conf.EnvironmentVariablesFor("app").MustGet())
	})

	t.Run("should return an empty monad if no environment variables are defined at all", func(t *testing.T) {
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		testutil.IsFalse(t, conf.EnvironmentVariablesFor("app").HasValue())
	})

	t.Run("should have a subdomain equals to app name if env is production", func(t *testing.T) {
		conf, _ := app.ConfigSnapshotFor(domain.Production)

		testutil.Equals(t, "my-app", conf.SubDomain())
	})

	t.Run("should have a subdomain suffixed by the env if not production", func(t *testing.T) {
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		testutil.Equals(t, "my-app-staging", conf.SubDomain())
	})

	t.Run("should expose a project name", func(t *testing.T) {
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		testutil.Equals(t, "my-app-staging", conf.ProjectName())
	})
}
