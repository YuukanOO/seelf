package reconfigure_target_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/reconfigure_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_ReconfigureTarget(t *testing.T) {
	sut := func(existingTargets ...*domain.Target) bus.RequestHandler[bus.UnitType, reconfigure_target.Command] {
		store := memory.NewTargetsStore(existingTargets...)
		return reconfigure_target.Handler(store, store)
	}

	t.Run("should returns an err if the target does not exist", func(t *testing.T) {
		uc := sut()

		_, err := uc(context.Background(), reconfigure_target.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should force the reconfiguration of the target", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), nil)

		uc := sut(&target)

		_, err := uc(context.Background(), reconfigure_target.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		changed := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		testutil.Equals(t, domain.TargetStatusConfiguring, changed.State.Status())
	})
}
