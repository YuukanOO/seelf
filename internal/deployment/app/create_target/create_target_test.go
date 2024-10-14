package create_target_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_target"
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

func Test_CreateTarget(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, create_target.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return create_target.Handler(context.TargetsStore, context.TargetsStore, &dummyProvider{}), context.Context, context.Dispatcher
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		_, err := handler(ctx, create_target.Command{})

		assert.ValidationError(t, validate.FieldErrors{
			"name": strings.ErrRequired,
		}, err)
	})

	t.Run("should require a unique url and config", func(t *testing.T) {
		var config = fixture.ProviderConfig()
		user := authfixture.User()
		target := fixture.Target(
			fixture.WithTargetCreatedBy(user.ID()),
			fixture.WithProviderConfig(config),
		)
		assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true)))

		handler, ctx, _ := arrange(t, fixture.WithUsers(&user), fixture.WithTargets(&target))

		_, err := handler(ctx, create_target.Command{
			Name:     "target",
			Url:      monad.Value("http://example.com"),
			Provider: config,
		})

		assert.ValidationError(t, validate.FieldErrors{
			"url":         domain.ErrUrlAlreadyTaken,
			config.Kind(): domain.ErrConfigAlreadyTaken,
		}, err)
	})

	t.Run("should require a valid provider config", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		_, err := handler(ctx, create_target.Command{
			Name: "target",
		})

		assert.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should allow multiple manual targets to co-exists", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, ctx, dispatcher := arrange(t, fixture.WithUsers(&user), fixture.WithTargets(&target))

		id, err := handler(ctx, create_target.Command{
			Name:     "target-one",
			Provider: fixture.ProviderConfig(),
		})

		assert.Nil(t, err)
		assert.NotZero(t, id)

		id, err = handler(ctx, create_target.Command{
			Name:     "target-two",
			Provider: fixture.ProviderConfig(),
		})

		assert.Nil(t, err)
		assert.NotZero(t, id)

		assert.HasLength(t, 2, dispatcher.Signals())
	})

	t.Run("should create a new target", func(t *testing.T) {
		var config = fixture.ProviderConfig()
		user := authfixture.User()
		handler, ctx, dispatcher := arrange(t, fixture.WithUsers(&user))

		id, err := handler(ctx, create_target.Command{
			Name:     "target",
			Url:      monad.Value("http://example.com"),
			Provider: config,
		})

		assert.Nil(t, err)
		assert.NotZero(t, id)
		assert.HasLength(t, 3, dispatcher.Signals())

		created := assert.Is[domain.TargetCreated](t, dispatcher.Signals()[0])
		assert.DeepEqual(t, domain.TargetCreated{
			ID:          domain.TargetID(id),
			Name:        "target",
			State:       created.State,
			Entrypoints: make(domain.TargetEntrypoints),
			Provider:    config, // Since the mock returns the config "as is"
			Created:     shared.ActionFrom(user.ID(), assert.NotZero(t, created.Created.At())),
		}, created)

		urlChanged := assert.Is[domain.TargetUrlChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.TargetUrlChanged{
			ID:  domain.TargetID(id),
			Url: must.Panic(domain.UrlFrom("http://example.com")),
		}, urlChanged)

		assert.Is[domain.TargetStateChanged](t, dispatcher.Signals()[2])
	})
}

type dummyProvider struct {
	domain.Provider
}

func (*dummyProvider) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	if payload == nil {
		return nil, domain.ErrNoValidProviderFound
	}

	return payload.(domain.ProviderConfig), nil
}
