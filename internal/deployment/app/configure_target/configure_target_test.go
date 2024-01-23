package configure_target_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app/configure_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_ConfigureTarget(t *testing.T) {
	sut := func(existingTargets ...*domain.Target) (bus.RequestHandler[bus.UnitType, configure_target.Command], *dummyProvider) {
		provider := &dummyProvider{}
		store := memory.NewTargetsStore(existingTargets...)
		return configure_target.Handler(store, store, provider), provider
	}

	t.Run("should fail silently if the target is not found", func(t *testing.T) {
		uc, provider := sut()

		_, err := uc(context.Background(), configure_target.Command{})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should returns early if the version is outdated", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		uc, provider := sut(&target)

		_, err := uc(context.Background(), configure_target.Command{
			ID:      string(target.ID()),
			Version: created.State.Version().Add(-1 * time.Second),
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should correctly mark the target as failed if the provider fails", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		uc, provider := sut(&target)
		providerErr := errors.New("some error")
		provider.err = providerErr

		_, err := uc(context.Background(), configure_target.Command{
			ID:      string(target.ID()),
			Version: created.State.Version(),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, provider.called)
		evt := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		testutil.Equals(t, domain.TargetStatusFailed, evt.State.Status())
		testutil.Equals(t, providerErr.Error(), evt.State.ErrCode().MustGet())
	})

	t.Run("should correctly mark the target as configured if everything is good", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		uc, provider := sut(&target)

		_, err := uc(context.Background(), configure_target.Command{
			ID:      string(target.ID()),
			Version: created.State.Version(),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, provider.called)
		evt := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		testutil.Equals(t, domain.TargetStatusReady, evt.State.Status())
	})
}

type dummyProvider struct {
	domain.Provider
	err    error
	called bool
}

func (d *dummyProvider) Setup(context.Context, domain.Target) error {
	d.called = true
	return d.err
}
