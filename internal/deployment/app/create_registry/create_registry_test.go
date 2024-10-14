package create_registry_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_registry"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

func Test_CreateRegistry(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, create_registry.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return create_registry.Handler(context.RegistriesStore, context.RegistriesStore), context.Context, context.Dispatcher
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		id, err := handler(ctx, create_registry.Command{})

		assert.Zero(t, id)
		assert.ValidationError(t, validate.FieldErrors{
			"name": strings.ErrRequired,
			"url":  domain.ErrInvalidUrl,
		}, err)
	})

	t.Run("should fail if the url is already taken", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(
			fixture.WithRegistryCreatedBy(user.ID()),
			fixture.WithUrl(must.Panic(domain.UrlFrom("http://example.com"))),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithRegistries(&registry),
		)

		id, err := handler(ctx, create_registry.Command{
			Name: "registry",
			Url:  "http://example.com",
		})

		assert.Zero(t, id)
		assert.ValidationError(t, validate.FieldErrors{
			"url": domain.ErrUrlAlreadyTaken,
		}, err)
	})

	t.Run("should create a new registry if everything is good", func(t *testing.T) {
		user := authfixture.User()
		handler, ctx, dispatcher := arrange(t, fixture.WithUsers(&user))

		id, err := handler(ctx, create_registry.Command{
			Name: "registry",
			Url:  "http://example.com",
			Credentials: monad.Value(create_registry.Credentials{
				Username: "user",
				Password: "password",
			}),
		})

		assert.NotZero(t, id)
		assert.Nil(t, err)
		assert.HasLength(t, 2, dispatcher.Signals())

		created := assert.Is[domain.RegistryCreated](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryCreated{
			ID:      domain.RegistryID(id),
			Name:    "registry",
			Url:     must.Panic(domain.UrlFrom("http://example.com")),
			Created: shared.ActionFrom(user.ID(), assert.NotZero(t, created.Created.At())),
		}, created)

		credentialsSet := assert.Is[domain.RegistryCredentialsChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.RegistryCredentialsChanged{
			ID:          domain.RegistryID(id),
			Credentials: domain.NewCredentials("user", "password"),
		}, credentialsSet)
	})
}
