package redeploy

import (
	"context"
	"errors"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnAppEnvChangedHandler(
	appsReader domain.AppsReader,
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		source, err := reader.GetLastDeployment(ctx, evt.ID, evt.Environment)

		if err != nil {
			// No deployment yet, nothing to do
			if errors.Is(err, apperr.ErrNotFound) {
				return nil
			}

			return err
		}

		app, err := appsReader.GetByID(ctx, evt.ID)

		if err != nil {
			return err
		}

		number, err := reader.GetNextDeploymentNumber(ctx, app.ID())

		if err != nil {
			return err
		}

		deployment, err := app.Redeploy(source, number, auth.CurrentUser(ctx).MustGet())

		// Could not redeploy the latest deployment, maybe because of a configuration change,
		// just skip it (for example, trying to redeploy a git deployment but the vcs is now missing)
		if err != nil {
			return nil
		}

		return writer.Write(ctx, &deployment)
	}
}
