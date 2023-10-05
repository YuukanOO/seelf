package cleanup

import (
	"context"

	deplcmd "github.com/YuukanOO/seelf/internal/deployment/app/command"
	depldomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs"
	"github.com/YuukanOO/seelf/pkg/log"
)

const jobName = "deployment:cleanup-app"

// Creates a new deployment job for the given deployment id.
func Queue(id depldomain.AppID) command.QueueCommand {
	return command.QueueCommand{
		Name:    jobName,
		Payload: string(id),
	}
}

type handler struct {
	logger  log.Logger
	cleanup func(context.Context, deplcmd.CleanupAppCommand) error
}

func New(logger log.Logger, cleanup func(context.Context, deplcmd.CleanupAppCommand) error) jobs.Handler {
	return &handler{
		logger:  logger,
		cleanup: cleanup,
	}
}

func (*handler) JobName() string {
	return jobName
}

func (h *handler) Process(ctx context.Context, job domain.Job) error {
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
