package update_registry_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_registry"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateRegistry(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, update_registry.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return update_registry.Handler(context.RegistriesStore, context.RegistriesStore), context.Dispatcher
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), update_registry.Command{
			Url: monad.Value("not an url"),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"url": domain.ErrInvalidUrl,
		}, err)
	})

	t.Run("should require an existing registry", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), update_registry.Command{
			Url: monad.Value("http://example.com"),
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should rename a registry", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		handler, dispatcher := arrange(t, fixture.WithUsers(&user), fixture.WithRegistries(&registry))

		id, err := handler(context.Background(), update_registry.Command{
			ID:   string(registry.ID()),
			Name: monad.Value("new-name"),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(registry.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		renamed := assert.Is[domain.RegistryRenamed](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryRenamed{
			ID:   registry.ID(),
			Name: "new-name",
		}, renamed)
	})

	t.Run("should require a unique url when updating it", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(
			fixture.WithRegistryCreatedBy(user.ID()),
			fixture.WithUrl(must.Panic(domain.UrlFrom("http://example.com"))),
		)
		otherRegistry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		handler, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithRegistries(&registry, &otherRegistry),
		)

		_, err := handler(context.Background(), update_registry.Command{
			ID:  string(otherRegistry.ID()),
			Url: monad.Value("http://example.com"),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"url": domain.ErrUrlAlreadyTaken,
		}, err)
	})

	t.Run("should update the url if its good", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		handler, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithRegistries(&registry),
		)

		id, err := handler(context.Background(), update_registry.Command{
			ID:  string(registry.ID()),
			Url: monad.Value("http://localhost:5000"),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(registry.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.RegistryUrlChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryUrlChanged{
			ID:  registry.ID(),
			Url: must.Panic(domain.UrlFrom("http://localhost:5000")),
		}, changed)
	})

	t.Run("should be able to add credentials", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		handler, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithRegistries(&registry),
		)

		id, err := handler(context.Background(), update_registry.Command{
			ID: string(registry.ID()),
			Credentials: monad.PatchValue(update_registry.Credentials{
				Username: "user",
				Password: monad.Value("password"),
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(registry.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.RegistryCredentialsChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryCredentialsChanged{
			ID:          registry.ID(),
			Credentials: domain.NewCredentials("user", "password"),
		}, changed)
	})

	t.Run("should be able to update only the credentials username", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		registry.UseAuthentication(domain.NewCredentials("user", "password"))
		handler, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithRegistries(&registry),
		)

		id, err := handler(context.Background(), update_registry.Command{
			ID: string(registry.ID()),
			Credentials: monad.PatchValue(update_registry.Credentials{
				Username: "new-user",
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(registry.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.RegistryCredentialsChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryCredentialsChanged{
			ID:          registry.ID(),
			Credentials: domain.NewCredentials("new-user", "password"),
		}, changed)
	})

	t.Run("should be able to remove authentication", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		registry.UseAuthentication(domain.NewCredentials("user", "password"))
		handler, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithRegistries(&registry),
		)

		id, err := handler(context.Background(), update_registry.Command{
			ID:          string(registry.ID()),
			Credentials: monad.Nil[update_registry.Credentials](),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(registry.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		removed := assert.Is[domain.RegistryCredentialsRemoved](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryCredentialsRemoved{
			ID: registry.ID(),
		}, removed)
	})
}
