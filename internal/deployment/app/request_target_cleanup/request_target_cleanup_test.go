package request_target_cleanup_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_target_cleanup"
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

func Test_RequestTargetCleanup(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")

	sut := func(existing initialData) bus.RequestHandler[bus.UnitType, request_target_cleanup.Command] {
		targetsStore := memory.NewTargetsStore(existing.targets...)
		appsStore := memory.NewAppsStore(existing.apps...)
		return request_target_cleanup.Handler(targetsStore, targetsStore, appsStore)
	}

	t.Run("should returns an error if the target does not exist", func(t *testing.T) {
		uc := sut(initialData{})

		_, err := uc(ctx, request_target_cleanup.Command{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should returns an error if the target has still apps using it", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true),
			domain.NewProviderConfigRequirement(dummyProviderConfig{}, true), "uid"))
		target.Configured(target.CurrentVersion(), nil, nil)
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true), "uid"))

		uc := sut(initialData{
			targets: []*domain.Target{&target},
			apps:    []*domain.App{&app},
		})

		_, err := uc(ctx, request_target_cleanup.Command{
			ID: string(target.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetInUse, err)
	})

	t.Run("should correctly mark the target for cleanup", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true),
			domain.NewProviderConfigRequirement(dummyProviderConfig{}, true), "uid"))
		target.Configured(target.CurrentVersion(), nil, nil)

		uc := sut(initialData{
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, request_target_cleanup.Command{
			ID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &target, 3)
		evt := testutil.EventIs[domain.TargetCleanupRequested](t, &target, 2)
		testutil.Equals(t, target.ID(), evt.ID)
	})
}

type dummyProviderConfig struct {
	domain.ProviderConfig
}
