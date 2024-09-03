package configure_target_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/configure_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
)

func Test_ConfigureTarget(t *testing.T) {

	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, configure_target.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return configure_target.Handler(context.TargetsStore, context.TargetsStore, provider), context.Dispatcher
	}

	t.Run("should fail silently if the target is not found", func(t *testing.T) {
		var provider dummyProvider
		handler, _ := arrange(t, &provider)

		_, err := handler(context.Background(), configure_target.Command{})

		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should returns early if the version is outdated", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, dispatcher := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(context.Background(), configure_target.Command{
			ID:      string(target.ID()),
			Version: target.CurrentVersion().Add(-1 * time.Second),
		})

		assert.Nil(t, err)
		assert.HasLength(t, 0, dispatcher.Signals())
		assert.False(t, provider.called)
	})

	t.Run("should correctly mark the target as failed if the provider fails", func(t *testing.T) {
		providerErr := errors.New("some error")
		provider := dummyProvider{err: providerErr}
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, dispatcher := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(context.Background(), configure_target.Command{
			ID:      string(target.ID()),
			Version: target.CurrentVersion(),
		})

		assert.Nil(t, err)
		assert.True(t, provider.called)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.TargetStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetStatusFailed, changed.State.Status())
		assert.Equal(t, providerErr.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should correctly mark the target as configured if everything is good", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, dispatcher := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(context.Background(), configure_target.Command{
			ID:      string(target.ID()),
			Version: target.CurrentVersion(),
		})

		assert.Nil(t, err)
		assert.True(t, provider.called)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.TargetStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetStatusReady, changed.State.Status())
		assert.Equal(t, target.CurrentVersion(), changed.State.LastReadyVersion().MustGet())
	})
}

type dummyProvider struct {
	domain.Provider
	err    error
	called bool
}

func (d *dummyProvider) Setup(context.Context, domain.Target) (domain.TargetEntrypointsAssigned, error) {
	d.called = true
	return nil, d.err
}
