package deploy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	deplcmd "github.com/YuukanOO/seelf/internal/deployment/app/command"
	depldomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
)

const JobName = "deployment:deploy"

// Creates a new deployment job for the given deployment id.
func Queue(evt depldomain.DeploymentCreated) command.QueueCommand {
	id := evt.ID

	return command.QueueCommand{
		Name:       JobName,
		Payload:    fmt.Sprintf("%s:%d", id.AppID(), id.DeploymentNumber()),
		DedupeName: monad.Value(fmt.Sprintf("%s:%s", JobName, evt.Config.ProjectName())),
	}
}

type handler struct {
	logger log.Logger
	deploy func(context.Context, deplcmd.DeployCommand) error
}

func New(logger log.Logger, deploy func(context.Context, deplcmd.DeployCommand) error) jobs.Handler {
	return &handler{
		logger: logger,
		deploy: deploy,
	}
}

func (*handler) JobName() string {
	return JobName
}

func (h *handler) Process(ctx context.Context, job domain.Job) error {
	parts := strings.Split(job.Payload(), ":")
	appid := parts[0]
	number, _ := strconv.Atoi(parts[1])

	// Here the error is not given back to the worker because if it fails, the information
	// is already in the associated Deployment. The only exception is for sql errors.
	if err := h.deploy(ctx, deplcmd.DeployCommand{
		AppID:            appid,
		DeploymentNumber: number,
	}); err != nil {
		h.logger.Errorw("deploy job has failed",
			"error", err,
			"appid", appid,
			"deployment", number,
		)
	}

	return nil
}
