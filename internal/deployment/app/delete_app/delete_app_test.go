package delete_app_test

import (
	"context"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func DeleteApp(t *testing.T) {
	ctx := context.Background()
	logger, _ := log.NewLogger()

	sut := func(initialApps ...*domain.App) bus.RequestHandler[bus.UnitType, delete_app.Command] {
		opts := config.Default(config.WithTestDefaults())
		appsStore := memory.NewAppsStore(initialApps...)
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return delete_app.Handler(appsStore, appsStore, artifactManager)
	}

	t.Run("should fail silently if the application does not exist anymore", func(t *testing.T) {
		uc := sut()

		r, err := uc(ctx, delete_app.Command{
			ID: "some-id",
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should fail if the application cleanup has not been requested", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "uid"))
		uc := sut(&app)

		r, err := uc(ctx, delete_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "uid"))
		app.RequestCleanup("uid")

		uc := sut(&app)

		r, err := uc(ctx, delete_app.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppDeleted](t, &app, 2)
	})
}
