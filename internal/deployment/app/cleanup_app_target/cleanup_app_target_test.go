package cleanup_app_target_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_CleanupAppTarget(t *testing.T) {
	ctx := context.Background()

	sut := func(existingTargets ...*domain.Target) (bus.RequestHandler[bus.UnitType, cleanup_app_target.Command], *dummyProvider) {
		targetsStore := memory.NewTargetsStore(existingTargets...)
		provider := &dummyProvider{}
		return cleanup_app_target.Handler(targetsStore, provider), provider
	}

	t.Run("should fail silently if the target does not exist anymore", func(t *testing.T) {
		uc, provider := sut()

		r, err := uc(ctx, cleanup_app_target.Command{})

		testutil.Equals(t, bus.Ignore(apperr.ErrNotFound), err)
		testutil.Equals(t, bus.Unit, r)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the target is configuring", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, nil, true, "uid"))

		uc, provider := sut(&target)

		_, err := uc(ctx, cleanup_app_target.Command{
			TargetID: string(target.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target is being deleted", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)
		target.RequestDelete(0, "uid")

		uc, provider := sut(&target)

		_, err := uc(ctx, cleanup_app_target.Command{
			TargetID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the target configuration failed but has once been reachable", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)
		target.Reconfigure()
		changed := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		target.Configured(changed.State.Version(), errors.New("configuration-failed"))

		uc, provider := sut(&target)

		_, err := uc(ctx, cleanup_app_target.Command{
			TargetID: string(target.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target configuration failed and has never been reachable", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), errors.New("configuration-failed"))

		uc, provider := sut(&target)

		_, err := uc(ctx, cleanup_app_target.Command{
			TargetID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target is ready", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)

		uc, provider := sut(&target)

		_, err := uc(ctx, cleanup_app_target.Command{
			TargetID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, provider.called)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) Cleanup(context.Context, domain.AppID, domain.Target, domain.Environment) error {
	d.called = true
	return nil
}
