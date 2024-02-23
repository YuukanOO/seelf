package delete_target_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/delete_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type initialData struct {
	targets []*domain.Target
	apps    []*domain.App
}

func Test_DeleteTarget(t *testing.T) {
	sut := func(existing initialData) bus.RequestHandler[bus.UnitType, delete_target.Command] {
		targetsStore := memory.NewTargetsStore(existing.targets...)
		appsStore := memory.NewAppsStore(existing.apps...)
		return delete_target.Handler(targetsStore, targetsStore, appsStore)
	}

	t.Run("should returns an error if the target does not exist", func(t *testing.T) {
		uc := sut(initialData{})

		_, err := uc(context.Background(), delete_target.Command{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should returns an error if the target has still apps using it", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, dummyProviderConfig{}, true, "uid"))
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()),
			domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))

		uc := sut(initialData{
			targets: []*domain.Target{&target},
			apps:    []*domain.App{&app},
		})

		_, err := uc(context.Background(), delete_target.Command{
			ID: string(target.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetUsed, err)
	})

	t.Run("should correctly delete the target", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, dummyProviderConfig{}, true, "uid"))

		uc := sut(initialData{
			targets: []*domain.Target{&target},
		})

		_, err := uc(context.Background(), delete_target.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &target, 2)
		evt := testutil.EventIs[domain.TargetDeleted](t, &target, 1)
		testutil.Equals(t, target.ID(), evt.ID)
	})
}

type dummyProviderConfig struct {
	domain.ProviderConfig
}
