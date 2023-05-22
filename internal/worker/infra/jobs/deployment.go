package jobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	deplcmd "github.com/YuukanOO/seelf/internal/deployment/app/command"
	depldomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra"
	"github.com/YuukanOO/seelf/pkg/log"
)

const deploymentJobName = "deployment:deploy"

// Creates a new deployment job for the given deployment id.
func Deployment(id depldomain.DeploymentID) command.QueueCommand {
	return command.QueueCommand{
		Name:    deploymentJobName,
		Payload: fmt.Sprintf("%s:%d", id.AppID(), id.DeploymentNumber()),
	}
}

type deploymentHandler struct {
	logger log.Logger
	deploy func(context.Context, deplcmd.DeployCommand) error
}

func DeploymentHandler(logger log.Logger, deploy func(context.Context, deplcmd.DeployCommand) error) infra.JobsHandler {
	return &deploymentHandler{
		logger: logger,
		deploy: deploy,
	}
}

func (*deploymentHandler) JobName() string {
	return deploymentJobName
}

func (h *deploymentHandler) Process(ctx context.Context, job domain.Job) error {
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
