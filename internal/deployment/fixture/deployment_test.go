package fixture_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Deployment(t *testing.T) {
	t.Run("should be able to create a new deployment", func(t *testing.T) {
		deployment := fixture.Deployment()

		assert.NotZero(t, deployment.ID())
		assert.Equal(t, domain.Production, deployment.Config().Environment())
	})

	t.Run("should be able to create a new deployment from a given app", func(t *testing.T) {
		app := fixture.App()
		deployment := fixture.Deployment(fixture.FromApp(app))

		created := assert.EventIs[domain.DeploymentCreated](t, &deployment, 0)
		assert.Equal(t, app.ID(), created.ID.AppID())
	})

	t.Run("should be able to create a new deployment requested by a given user id", func(t *testing.T) {
		deployment := fixture.Deployment(fixture.WithDeploymentRequestedBy("uid"))

		created := assert.EventIs[domain.DeploymentCreated](t, &deployment, 0)
		assert.Equal(t, "uid", created.Requested.By())
	})

	t.Run("should be able to create a new deployment with a given source data", func(t *testing.T) {
		source := fixture.SourceData()
		deployment := fixture.Deployment(fixture.WithSourceData(source))

		created := assert.EventIs[domain.DeploymentCreated](t, &deployment, 0)
		assert.Equal(t, source, created.Source)
	})

	t.Run("should be able to create a new deployment with a given environment", func(t *testing.T) {
		deployment := fixture.Deployment(fixture.ForEnvironment(domain.Staging))

		created := assert.EventIs[domain.DeploymentCreated](t, &deployment, 0)
		assert.Equal(t, domain.Staging, created.Config.Environment())
	})
}

func Test_SourceData(t *testing.T) {
	t.Run("should be able to create a source data", func(t *testing.T) {
		source := fixture.SourceData()

		assert.Equal(t, "test", source.Kind())
		assert.False(t, source.NeedVersionControl())
	})

	t.Run("should be able to create a source data with version control needed", func(t *testing.T) {
		source := fixture.SourceData(fixture.WithVersionControlNeeded())

		assert.True(t, source.NeedVersionControl())
	})
}
