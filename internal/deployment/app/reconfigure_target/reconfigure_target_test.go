package reconfigure_target_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/reconfigure_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
)

func Test_ReconfigureTarget(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, reconfigure_target.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return reconfigure_target.Handler(context.TargetsStore, context.TargetsStore), context.Dispatcher
	}

	t.Run("should returns an error if the target does not exist", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), reconfigure_target.Command{})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should returns an error if the target is already being configured", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, _ := arrange(t, fixture.WithUsers(&user), fixture.WithTargets(&target))

		_, err := handler(context.Background(), reconfigure_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
	})

	t.Run("should returns an error if the target is being deleted", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestDelete(false, user.ID()))
		handler, _ := arrange(t, fixture.WithUsers(&user), fixture.WithTargets(&target))

		_, err := handler(context.Background(), reconfigure_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetCleanupRequested, err)
	})

	t.Run("should reconfigure the target if everything is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		handler, dispatcher := arrange(t, fixture.WithUsers(&user), fixture.WithTargets(&target))

		_, err := handler(context.Background(), reconfigure_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.TargetStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetStatusConfiguring, changed.State.Status())
	})
}
