package artifact_test

import (
	"context"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_LocalArtifactManager(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	env := domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true)
	app := must.Panic(domain.NewApp("my-app", env, env, "some-uid"))
	depl := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid"))

	sut := func() domain.ArtifactManager {
		opts := config.Default(config.WithTestDefaults())

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return artifact.NewLocal(opts, logger)
	}

	t.Run("should correctly prepare a build directory", func(t *testing.T) {
		manager := sut()

		ctx, err := manager.PrepareBuild(context.Background(), depl)
		testutil.IsNil(t, err)
		testutil.IsNotNil(t, logger)

		defer ctx.Logger().Close()

		_, err = os.ReadDir(ctx.BuildDirectory())
		testutil.IsNil(t, err)
	})

	t.Run("should correctly cleanup an app directory", func(t *testing.T) {
		manager := sut()

		ctx, err := manager.PrepareBuild(context.Background(), depl)
		testutil.IsNil(t, err)

		ctx.Logger().Close() // Do not defer or else the directory will be locked

		err = manager.Cleanup(context.Background(), app.ID())
		testutil.IsNil(t, err)

		_, err = os.ReadDir(ctx.BuildDirectory())
		testutil.IsTrue(t, os.IsNotExist(err))
	})
}
