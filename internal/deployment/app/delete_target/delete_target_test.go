package delete_target_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/delete_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_DeleteTarget(t *testing.T) {
	ctx := context.Background()

	sut := func(existingTargets ...*domain.Target) (bus.RequestHandler[bus.UnitType, delete_target.Command], *dummyProvider) {
		targetsStore := memory.NewTargetsStore(existingTargets...)
		provider := &dummyProvider{}
		return delete_target.Handler(targetsStore, targetsStore, provider), provider
	}

	t.Run("should fail silently if the target does not exist anymore", func(t *testing.T) {
		uc, provider := sut()

		_, err := uc(ctx, delete_target.Command{})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the target has not been requested for cleanup", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))

		uc, provider := sut(&target)

		_, err := uc(ctx, delete_target.Command{
			ID: string(target.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetCleanupNeeded, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), nil, nil)
		testutil.IsNil(t, target.RequestCleanup(false, "uid"))

		uc, provider := sut(&target)

		_, err := uc(ctx, delete_target.Command{
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

func (d *dummyProvider) RemoveConfiguration(ctx context.Context, target domain.Target) error {
	d.called = true
	return nil
}
