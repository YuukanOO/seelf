package jobs

import (
	"context"

	deplcmd "github.com/YuukanOO/seelf/internal/deployment/app/command"
	depldomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra"
	"github.com/YuukanOO/seelf/pkg/log"
)

const cleanupAppJobName = "deployment:cleanup-app"

// Creates a new deployment job for the given deployment id.
func CleanupApp(id depldomain.AppID) command.QueueCommand {
	return command.QueueCommand{
		Name:    cleanupAppJobName,
		Payload: string(id),
	}
}

type cleanupAppHandler struct {
	logger  log.Logger
	cleanup func(context.Context, deplcmd.CleanupAppCommand) error
}

func CleanupAppHandler(logger log.Logger, cleanup func(context.Context, deplcmd.CleanupAppCommand) error) infra.JobsHandler {
	return &cleanupAppHandler{
		logger:  logger,
		cleanup: cleanup,
	}
}

func (*cleanupAppHandler) JobName() string {
	return cleanupAppJobName
}

func (h *cleanupAppHandler) Process(ctx context.Context, job domain.Job) error {
	appid := job.Payload()

	if err := h.cleanup(ctx, deplcmd.CleanupAppCommand{
		ID: appid,
	}); err != nil {
		h.logger.Errorw("cleanup job has failed",
			"error", err,
			"appid", appid,
		)
		return err
	}

	return nil
}
