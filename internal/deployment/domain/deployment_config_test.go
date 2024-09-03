package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Config(t *testing.T) {

	t.Run("could be created from an app", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production-target")
		config.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "prodadmin"},
		})
		app := fixture.App(fixture.WithAppName("my-app"), fixture.WithProductionConfig(config))
		conf, err := app.ConfigSnapshotFor(domain.Production)

		assert.Nil(t, err)
		assert.Equal(t, app.ID(), conf.AppID())
		assert.Equal(t, "my-app", conf.AppName())
		assert.Equal(t, domain.Production, conf.Environment())
		assert.Equal(t, config.Target(), conf.Target())
		assert.DeepEqual(t, config.Vars(), conf.Vars())
	})

	t.Run("should fail if env is not valid", func(t *testing.T) {
		app := fixture.App()
		_, err := app.ConfigSnapshotFor("invalid")

		assert.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
	})

	t.Run("should provide a way to retrieve environment variables for a service name", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production-target")
		config.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "prodadmin"},
		})
		app := fixture.App(fixture.WithAppName("my-app"), fixture.WithProductionConfig(config))
		conf, _ := app.ConfigSnapshotFor(domain.Production)

		assert.False(t, conf.EnvironmentVariablesFor("otherservice").HasValue())
		assert.True(t, conf.EnvironmentVariablesFor("app").HasValue())
		assert.DeepEqual(t, domain.EnvVars{
			"DEBUG": "false",
		}, conf.EnvironmentVariablesFor("app").MustGet())
	})

	t.Run("should return an empty monad if no environment variables are defined at all", func(t *testing.T) {
		app := fixture.App()
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		assert.False(t, conf.EnvironmentVariablesFor("app").HasValue())
	})

	t.Run("should generate a subdomain equals to app name if env is production", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		conf, _ := app.ConfigSnapshotFor(domain.Production)

		assert.Equal(t, "my-app", conf.SubDomain("app", true))
		assert.Equal(t, "db.my-app", conf.SubDomain("db", false))
	})

	t.Run("should generate a subdomain suffixed by the env if not production", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		assert.Equal(t, "my-app-staging", conf.SubDomain("app", true))
		assert.Equal(t, "db.my-app-staging", conf.SubDomain("db", false))
	})

	t.Run("should expose a unique project name", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		assert.Equal(t, fmt.Sprintf("my-app-staging-%s", strings.ToLower(string(app.ID()))), conf.ProjectName())
	})

	t.Run("should expose a unique image name for a service", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		assert.Equal(t, fmt.Sprintf("my-app-%s/app:staging", strings.ToLower(string(app.ID()))), conf.ImageName("app"))
	})

	t.Run("should expose a unique qualified name for a service", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		conf, _ := app.ConfigSnapshotFor(domain.Staging)

		assert.Equal(t, fmt.Sprintf("my-app-staging-%s-app", strings.ToLower(string(app.ID()))), conf.QualifiedName("app"))
	})
}
