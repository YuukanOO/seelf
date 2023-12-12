package cleanup

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	depldomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/types"
)

type (
	Request depldomain.AppCleanupRequested

	handler struct {
		logger log.Logger
		bus    bus.Bus
	}
)

func New(logger log.Logger, bus bus.Bus) jobs.Handler {
	return &handler{
		logger: logger,
		bus:    bus,
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

	if _, err := bus.Send(h.bus, ctx, cleanup_app.Command{
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
