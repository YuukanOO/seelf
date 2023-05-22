package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/collections"
)

// Mark all running deployments as failed with the given reason. This is mostly used
// when the server has crashed or has been hard resetted and some job does not ended
// correctly. They will need a redeploy.
func FailRunningDeployments(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
) func(context.Context, error) error {
	return func(ctx context.Context, reason error) error {
		deployments, err := reader.GetRunningDeployments(ctx)

		if err != nil {
			return err
		}

		for idx := range deployments {
			err = deployments[idx].HasEnded(nil, reason)

			if err != nil {
				return err
			}
		}

		return writer.Write(ctx, collections.ToPointers(deployments)...)
	}
}
