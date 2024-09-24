package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_ConfigSnapshot(t *testing.T) {

	t.Run("could be created", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production-target")
		config.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "prodadmin"},
		})

		app := fixture.App(fixture.WithAppName("my-app"), fixture.WithProductionConfig(config))
		deployment := fixture.Deployment(fixture.FromApp(app))
		conf := deployment.Config()

		assert.Equal(t, app.ID(), conf.AppID())
		assert.Equal(t, "my-app", conf.AppName())
		assert.Equal(t, domain.Production, conf.Environment())
		assert.Equal(t, config.Target(), conf.Target())
		assert.DeepEqual(t, config.Vars(), conf.Vars())
	})

	t.Run("should provide a way to retrieve environment variables for a service name", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production-target")
		config.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
			"db":  {"USERNAME": "prodadmin"},
		})

		app := fixture.App(fixture.WithAppName("my-app"), fixture.WithProductionConfig(config))
		deployment := fixture.Deployment(fixture.FromApp(app))
		conf := deployment.Config()

		assert.False(t, conf.EnvironmentVariablesFor("otherservice").HasValue())
		assert.True(t, conf.EnvironmentVariablesFor("app").HasValue())
		assert.DeepEqual(t, domain.EnvVars{
			"DEBUG": "false",
		}, conf.EnvironmentVariablesFor("app").MustGet())
	})

	t.Run("should return an empty monad if no environment variables are defined at all", func(t *testing.T) {
		app := fixture.App()
		deployment := fixture.Deployment(fixture.FromApp(app), fixture.ForEnvironment(domain.Staging))
		conf := deployment.Config()

		assert.False(t, conf.EnvironmentVariablesFor("app").HasValue())
	})

	t.Run("should expose a unique project name", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		deployment := fixture.Deployment(fixture.FromApp(app), fixture.ForEnvironment(domain.Staging))
		conf := deployment.Config()

		assert.Equal(t, fmt.Sprintf("my-app-staging-%s", strings.ToLower(string(app.ID()))), conf.ProjectName())
	})
}
