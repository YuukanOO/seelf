package update_target_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_target"
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

func Test_UpdateTarget(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, update_target.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return update_target.Handler(context.TargetsStore, context.TargetsStore, &dummyProvider{}), context.Dispatcher
	}

	t.Run("should fail if the target does not exist", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), update_target.Command{})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should fail if url or config are already taken", func(t *testing.T) {
		user := authfixture.User()
		config := fixture.ProviderConfig()
		targetOne := fixture.Target(
			fixture.WithTargetCreatedBy(user.ID()),
			fixture.WithTargetUrl(must.Panic(domain.UrlFrom("http://localhost"))),
			fixture.WithProviderConfig(config),
		)
		targetTwo := fixture.Target(
			fixture.WithTargetCreatedBy(user.ID()),
			fixture.WithTargetUrl(must.Panic(domain.UrlFrom("http://docker.localhost"))),
		)
		handler, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&targetOne, &targetTwo),
		)

		_, err := handler(context.Background(), update_target.Command{
			ID:       string(targetTwo.ID()),
			Provider: config,
			Url:      monad.Value("http://localhost"),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"url":         domain.ErrUrlAlreadyTaken,
			config.Kind(): domain.ErrConfigAlreadyTaken,
		}, err)
	})

	t.Run("should update the target if everything is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(
			fixture.WithTargetCreatedBy(user.ID()),
			fixture.WithProviderConfig(fixture.ProviderConfig(fixture.WithFingerprint("test"))),
		)
		handler, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)
		newConfig := fixture.ProviderConfig(fixture.WithFingerprint("test"))

		id, err := handler(context.Background(), update_target.Command{
			ID:       string(target.ID()),
			Name:     monad.Value("new name"),
			Provider: newConfig,
			Url:      monad.Value("http://docker.localhost"),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(target.ID()), id)
		assert.HasLength(t, 5, dispatcher.Signals())

		renamed := assert.Is[domain.TargetRenamed](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetRenamed{
			ID:   target.ID(),
			Name: "new name",
		}, renamed)

		urlChanged := assert.Is[domain.TargetUrlChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.TargetUrlChanged{
			ID:  target.ID(),
			Url: must.Panic(domain.UrlFrom("http://docker.localhost")),
		}, urlChanged)

		assert.Is[domain.TargetStateChanged](t, dispatcher.Signals()[2])

		providerChanged := assert.Is[domain.TargetProviderChanged](t, dispatcher.Signals()[3])
		assert.Equal(t, domain.TargetProviderChanged{
			ID:       target.ID(),
			Provider: newConfig,
		}, providerChanged)

		assert.Is[domain.TargetStateChanged](t, dispatcher.Signals()[4])
	})
}

type dummyProvider struct {
	domain.Provider
}

func (*dummyProvider) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	return payload.(domain.ProviderConfig), nil
}
