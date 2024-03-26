package cleanup_target_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_CleanupTarget(t *testing.T) {
	sut := func(existingTargets ...*domain.Target) (bus.RequestHandler[bus.UnitType, cleanup_target.Command], *dummyProvider) {
		targetsStore := memory.NewTargetsStore(existingTargets...)
		deploymentsStore := memory.NewDeploymentsStore()
		provider := &dummyProvider{}
		return cleanup_target.Handler(targetsStore, targetsStore, deploymentsStore, provider), provider
	}

	t.Run("should silently fail if the target does not exist anymore", func(t *testing.T) {
		uc, provider := sut()

		_, err := uc(context.Background(), cleanup_target.Command{})

		testutil.Equals(t, bus.Ignore(apperr.ErrNotFound), err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should skip the provider cleanup if the target has never been reachable", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), errors.New("some error"))
		target.RequestDelete(0, "uid")

		uc, provider := sut(&target)

		_, err := uc(context.Background(), cleanup_target.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.EventIs[domain.TargetDeleted](t, &target, 3)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed of the target can be safely deleted", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)
		target.RequestDelete(0, "uid")

		uc, provider := sut(&target)

		_, err := uc(context.Background(), cleanup_target.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.EventIs[domain.TargetDeleted](t, &target, 3)
		testutil.IsTrue(t, provider.called)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) CleanupTarget(context.Context, domain.Target, domain.TargetCleanupStrategy) error {
	d.called = true
	return nil
}
