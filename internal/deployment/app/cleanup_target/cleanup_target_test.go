package cleanup_target_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_CleanupTarget(t *testing.T) {
	sut := func(existingTargets ...*domain.Target) (bus.RequestHandler[bus.UnitType, cleanup_target.Command], *dummyProvider) {
		targetsStore := memory.NewTargetsStore(existingTargets...)
		deploymentsStore := memory.NewDeploymentsStore()
		provider := &dummyProvider{}
		return cleanup_target.Handler(targetsStore, deploymentsStore, provider), provider
	}

	t.Run("should silently fail if the target does not exist anymore", func(t *testing.T) {
		uc, provider := sut()

		_, err := uc(context.Background(), cleanup_target.Command{})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should skip the provider cleanup if the target is not reachable", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), errors.New("some error"))

		uc, provider := sut(&target)

		_, err := uc(context.Background(), cleanup_target.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target can be safely deleted", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), nil)

		uc, provider := sut(&target)

		_, err := uc(context.Background(), cleanup_target.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, provider.called)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) CleanupTarget(_ context.Context, _ domain.Target, s domain.CleanupStrategy) error {
	d.called = s != domain.CleanupStrategySkip
	return nil
}
