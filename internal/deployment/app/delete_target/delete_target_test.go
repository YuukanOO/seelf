package delete_target_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
)

func Test_DeleteTarget(t *testing.T) {

	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, delete_target.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return delete_target.Handler(context.TargetsStore, context.TargetsStore, provider), context.Dispatcher
	}

	t.Run("should fail silently if the target does not exist anymore", func(t *testing.T) {
		var provider dummyProvider
		handler, dispatcher := arrange(t, &provider)

		_, err := handler(context.Background(), delete_target.Command{})

		assert.Nil(t, err)
		assert.False(t, provider.called)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should fail if the target has not been requested for cleanup", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, dispatcher := arrange(t, &provider, fixture.WithUsers(&user), fixture.WithTargets(&target))

		_, err := handler(context.Background(), delete_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetCleanupNeeded, err)
		assert.False(t, provider.called)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestCleanup(false, user.ID()))
		handler, dispatcher := arrange(t, &provider, fixture.WithUsers(&user), fixture.WithTargets(&target))

		_, err := handler(context.Background(), delete_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.True(t, provider.called)
		assert.HasLength(t, 1, dispatcher.Signals())

		deleted := assert.Is[domain.TargetDeleted](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetDeleted{
			ID: target.ID(),
		}, deleted)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) RemoveConfiguration(ctx context.Context, target domain.Target) error {
	d.called = true
	return nil
}
