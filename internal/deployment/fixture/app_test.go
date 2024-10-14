package fixture_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_App(t *testing.T) {
	t.Run("should be able to build a random app", func(t *testing.T) {
		app := fixture.App()

		assert.NotZero(t, app.ID())
	})

	t.Run("should be able to build an app with a given name", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("foo"))

		created := assert.EventIs[domain.AppCreated](t, &app, 0)
		assert.Equal(t, "foo", created.Name)
	})

	t.Run("should be able to build an app created by a specific user id", func(t *testing.T) {
		app := fixture.App(fixture.WithAppCreatedBy("uid"))

		created := assert.EventIs[domain.AppCreated](t, &app, 0)
		assert.Equal(t, "uid", created.Created.By())
	})

	t.Run("should be able to build an app with given production and staging configuration", func(t *testing.T) {
		production := domain.NewEnvironmentConfig("production_id")
		staging := domain.NewEnvironmentConfig("staging_id")
		app := fixture.App(fixture.WithEnvironmentConfig(production, staging))

		created := assert.EventIs[domain.AppCreated](t, &app, 0)
		assert.DeepEqual(t, production, created.Production)
		assert.DeepEqual(t, staging, created.Staging)
	})

	t.Run("should be able to build an app with given production configuration", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production_id")
		app := fixture.App(fixture.WithProductionConfig(config))

		created := assert.EventIs[domain.AppCreated](t, &app, 0)
		assert.DeepEqual(t, config, created.Production)
		assert.NotEqual(t, config.Target(), created.Staging.Target())
	})
}
