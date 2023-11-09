package cleanup

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	depldomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/types"
)

type (
	Request depldomain.AppCleanupRequested

	handler struct {
		logger  log.Logger
		cleanup func(context.Context, command.CleanupAppCommand) error
	}
)

func New(logger log.Logger, cleanup func(context.Context, command.CleanupAppCommand) error) jobs.Handler {
	return &handler{
		logger:  logger,
		cleanup: cleanup,
	}
}

func (*handler) CanPrepare(data any) bool            { return types.Is[Request](data) }
func (*handler) CanProcess(data domain.JobData) bool { return types.Is[Data](data) }

func (h *handler) Prepare(payload any) (domain.JobData, monad.Maybe[string], error) {
	req, ok := payload.(Request)

	if !ok {
		return nil, monad.None[string](), domain.ErrInvalidPayload
	}

	return Data(req.ID), monad.None[string](), nil
}

func (h *handler) Process(ctx context.Context, job domain.Job) error {
	data, ok := job.Data().(Data)

	if !ok {
		return domain.ErrInvalidPayload
	}

	if err := h.cleanup(ctx, command.CleanupAppCommand{
		ID: string(data),
	}); err != nil {
		h.logger.Errorw("cleanup job has failed",
			"error", err,
			"appid", data,
		)
		return err
	}

	return nil
}
