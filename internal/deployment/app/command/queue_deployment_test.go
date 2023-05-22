package command_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

type options struct{}

func (options) LogsDir() string { return "logs" }
func (options) AppsDir() string { return "apps" }
func (options) Execute(d domain.DeploymentTemplateData) string {
	return filepath.Join(strconv.Itoa(int(d.Number)), string(d.Environment))
}

func Test_QueueDeployment(t *testing.T) {
	opts := options{}
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := domain.NewApp("my-app", "some-uid")
	appsStore := memory.NewAppsStore(app)

	queue := func() func(ctx context.Context, cmd command.QueueDeploymentCommand) (int, error) {
		deploymentsStore := memory.NewDeploymentsStore()
		return command.QueueDeployment(appsStore, deploymentsStore, deploymentsStore, raw.New(opts), opts)
	}

	t.Run("should fail if payload is empty", func(t *testing.T) {
		uc := queue()
		_, err := uc(ctx, command.QueueDeploymentCommand{
			AppID:       string(app.ID()),
			Environment: "production",
		})

		testutil.ErrorIs(t, domain.ErrInvalidTriggerPayload, err)
	})

	t.Run("should fail if no environment has been given", func(t *testing.T) {
		uc := queue()
		_, err := uc(ctx, command.QueueDeploymentCommand{
			AppID: string(app.ID()),
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)

		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidEnvironmentName, validationErr.Fields["environment"])
	})

	t.Run("should fail if the app does not exist", func(t *testing.T) {
		uc := queue()
		_, err := uc(ctx, command.QueueDeploymentCommand{
			AppID:       "does-not-exist",
			Environment: "production",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		uc := queue()
		r, err := uc(ctx, command.QueueDeploymentCommand{
			AppID:       string(app.ID()),
			Environment: "production",
			Payload:     "some-payload",
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 1, r)
	})
}
