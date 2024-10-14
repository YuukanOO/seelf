package fixture_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Target(t *testing.T) {
	t.Run("should be able to create a target", func(t *testing.T) {
		target := fixture.Target()

		assert.NotZero(t, target.ID())
	})

	t.Run("should be able to create a target with a given name", func(t *testing.T) {
		target := fixture.Target(fixture.WithTargetName("name"))

		created := assert.EventIs[domain.TargetCreated](t, &target, 0)
		assert.Equal(t, "name", created.Name)
	})

	t.Run("should be able to create a target with a given user id", func(t *testing.T) {
		target := fixture.Target(fixture.WithTargetCreatedBy("id"))

		created := assert.EventIs[domain.TargetCreated](t, &target, 0)
		assert.Equal(t, "id", created.Created.By())
	})

	t.Run("should be able to create a target with a given provider config", func(t *testing.T) {
		config := fixture.ProviderConfig()
		target := fixture.Target(fixture.WithProviderConfig(config))

		created := assert.EventIs[domain.TargetCreated](t, &target, 0)
		assert.DeepEqual(t, config, created.Provider)
	})
}

func Test_ProviderConfig(t *testing.T) {
	t.Run("should be able to create a provider config", func(t *testing.T) {
		config := fixture.ProviderConfig()

		assert.NotZero(t, config.Fingerprint())
		assert.NotZero(t, config.Kind())
	})

	t.Run("should be able to create a provider config with a given fingerprint", func(t *testing.T) {
		config := fixture.ProviderConfig(fixture.WithFingerprint("fingerprint"))

		assert.Equal(t, "fingerprint", config.Fingerprint())
	})

	t.Run("should be able to create a provider config with a given kind", func(t *testing.T) {
		config := fixture.ProviderConfig(fixture.WithKind("kind"))

		assert.Equal(t, "kind", config.Kind())
	})

	t.Run("should be able to create a provider config with a given data", func(t *testing.T) {
		one := fixture.ProviderConfig(
			fixture.WithKind("kind"),
			fixture.WithFingerprint("fingerprint"),
			fixture.WithData("data"))
		two := fixture.ProviderConfig(
			fixture.WithKind("kind"),
			fixture.WithFingerprint("fingerprint"),
			fixture.WithData("data"))
		three := fixture.ProviderConfig()

		assert.True(t, one.Equals(two))
		assert.False(t, one.Equals(three))
		assert.False(t, two.Equals(three))
	})
}
