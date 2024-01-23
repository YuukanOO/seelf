package queue_deployment_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/queue_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_QueueDeployment(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := must.Panic(domain.NewApp("my-app",
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "some-uid"))
	appsStore := memory.NewAppsStore(&app)

	sut := func() bus.RequestHandler[int, queue_deployment.Command] {
		deploymentsStore := memory.NewDeploymentsStore()
		return queue_deployment.Handler(appsStore, deploymentsStore, deploymentsStore, raw.New())
	}

	t.Run("should fail if payload is empty", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, queue_deployment.Command{
			AppID:       string(app.ID()),
			Environment: "production",
		})

		testutil.ErrorIs(t, domain.ErrInvalidSourcePayload, err)
		testutil.Equals(t, 0, num)
	})

	t.Run("should fail if no environment has been given", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, queue_deployment.Command{
			AppID: string(app.ID()),
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		testutil.Equals(t, 0, num)

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidEnvironmentName, validationErr["environment"])
	})

	t.Run("should fail if the app does not exist", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, queue_deployment.Command{
			AppID:       "does-not-exist",
			Environment: "production",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, 0, num)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, queue_deployment.Command{
			AppID:       string(app.ID()),
			Environment: "production",
			Source:      "some-payload",
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 1, num)
	})
}
