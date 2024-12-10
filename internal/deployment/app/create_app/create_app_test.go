package create_app_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

func Test_CreateApp(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, create_app.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return create_app.Handler(context.AppsStore, context.AppsStore), context.Context, context.Dispatcher
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		id, err := handler(ctx, create_app.Command{})

		assert.Zero(t, id)
		assert.ValidationError(t, validate.FieldErrors{
			"name":              domain.ErrInvalidAppName,
			"production.target": strings.ErrRequired,
			"staging.target":    strings.ErrRequired,
		}, err)
	})

	t.Run("should fail if the name is already taken", func(t *testing.T) {
		user := authfixture.User()
		productionTarget := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		stagingTarget := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		existingApp := fixture.App(fixture.WithAppName("my-app"),
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(productionTarget.ID()),
				domain.NewEnvironmentConfig(stagingTarget.ID()),
			))
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&productionTarget, &stagingTarget),
			fixture.WithApps(&existingApp),
		)

		id, err := handler(ctx, create_app.Command{
			Name: "my-app",
			Production: create_app.EnvironmentConfig{
				Target: string(productionTarget.ID()),
			},
			Staging: create_app.EnvironmentConfig{
				Target: string(stagingTarget.ID()),
			},
		})

		assert.Zero(t, id)
		assert.ValidationError(t, validate.FieldErrors{
			"production.target": domain.ErrAppNameAlreadyTaken,
			"staging.target":    domain.ErrAppNameAlreadyTaken,
		}, err)
	})

	t.Run("should fail if provided targets does not exists", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		id, err := handler(ctx, create_app.Command{
			Name: "my-app",
			Production: create_app.EnvironmentConfig{
				Target: "production-target",
			},
			Staging: create_app.EnvironmentConfig{
				Target: "staging-target",
			},
		})

		assert.Zero(t, id)
		assert.ValidationError(t, validate.FieldErrors{
			"production.target": apperr.ErrNotFound,
			"staging.target":    apperr.ErrNotFound,
		}, err)
	})

	t.Run("should create a new app if everything is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		id, err := handler(ctx, create_app.Command{
			Name: "my-app",
			Production: create_app.EnvironmentConfig{
				Target: string(target.ID()),
			},
			Staging: create_app.EnvironmentConfig{
				Target: string(target.ID()),
			},
			VersionControl: monad.Value(create_app.VersionControl{
				Url:   "https://somewhere.git",
				Token: monad.Value("some-token"),
			}),
		})

		assert.Nil(t, err)
		assert.NotZero(t, id)
		assert.HasLength(t, 2, dispatcher.Signals())

		created := assert.Is[domain.AppCreated](t, dispatcher.Signals()[0])
		assert.DeepEqual(t, domain.AppCreated{
			ID:         domain.AppID(id),
			Name:       "my-app",
			Production: created.Production,
			Staging:    created.Staging,
			Created:    shared.ActionFrom(user.ID(), assert.NotZero(t, created.Created.At())),
			History: domain.AppTargetHistory{
				domain.Production: []domain.TargetID{created.Production.Target()},
				domain.Staging:    []domain.TargetID{created.Staging.Target()},
			},
		}, created)
		assert.Equal(t, target.ID(), created.Production.Target())
		assert.Equal(t, target.ID(), created.Staging.Target())

		versionControlConfigured := assert.Is[domain.AppVersionControlConfigured](t, dispatcher.Signals()[1])
		assert.Equal(t, created.ID, versionControlConfigured.ID)
		assert.Equal(t, "https://somewhere.git", versionControlConfigured.Config.Url().String())
		assert.Equal(t, "some-token", versionControlConfigured.Config.Token().Get(""))
	})
}
