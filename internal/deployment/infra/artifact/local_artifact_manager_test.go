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
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_LocalArtifactManager(t *testing.T) {
	logger, _ := log.NewLogger()

	sut := func() domain.ArtifactManager {
		opts := config.Default(config.WithTestDefaults())

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return artifact.NewLocal(opts, logger)
	}

	t.Run("should correctly prepare a build directory", func(t *testing.T) {
		app := domain.NewApp("my-app", "some-uid")
		depl, _ := app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid")
		manager := sut()

		dir, logger, err := manager.PrepareBuild(context.Background(), depl)
		testutil.IsNil(t, err)
		testutil.IsNotNil(t, logger)

		defer logger.Close()

		_, err = os.ReadDir(dir)
		testutil.IsNil(t, err)
	})

	t.Run("should correctly cleanup an app directory", func(t *testing.T) {
		app := domain.NewApp("my-app", "some-uid")
		depl, _ := app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid")
		manager := sut()

		dir, logger, err := manager.PrepareBuild(context.Background(), depl)
		testutil.IsNil(t, err)

		logger.Close() // Do not defer or else the directory will be locked

		err = manager.Cleanup(context.Background(), app)
		testutil.IsNil(t, err)

		_, err = os.ReadDir(dir)
		testutil.IsTrue(t, os.IsNotExist(err))
	})
}
