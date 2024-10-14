package fixture_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Registry(t *testing.T) {
	t.Run("should be able to create a random registry", func(t *testing.T) {
		registry := fixture.Registry()

		assert.NotZero(t, registry.ID())
	})

	t.Run("should be able to create a registry with a given name", func(t *testing.T) {
		registry := fixture.Registry(fixture.WithRegistryName("my-registry"))

		created := assert.EventIs[domain.RegistryCreated](t, &registry, 0)
		assert.Equal(t, "my-registry", created.Name)
	})

	t.Run("should be able to create a registry created by a given user id", func(t *testing.T) {
		registry := fixture.Registry(fixture.WithRegistryCreatedBy("uid"))

		created := assert.EventIs[domain.RegistryCreated](t, &registry, 0)
		assert.Equal(t, "uid", created.Created.By())
	})

	t.Run("should be able to create a registry with a given url", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("https://my-registry.com"))
		registry := fixture.Registry(fixture.WithUrl(url))

		created := assert.EventIs[domain.RegistryCreated](t, &registry, 0)
		assert.Equal(t, url, created.Url)
	})
}
