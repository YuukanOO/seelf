package jobs

import (
	"context"
	"fmt"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
)

type (
	Handler interface {
		domain.Handler
		CanPrepare(any) bool
		CanProcess(domain.JobData) bool
	}

	facade struct {
		handlers []Handler
	}
)

func NewFacade(handlers ...Handler) domain.Handler {
	return &facade{handlers}
}

func (w *facade) Prepare(payload any) (domain.JobData, monad.Maybe[string], error) {
	for _, handler := range w.handlers {
		if handler.CanPrepare(payload) {
			return handler.Prepare(payload)
		}
	}

	return nil, monad.None[string](), domain.ErrNoValidHandlerFound
}

func (w *facade) Process(ctx context.Context, job domain.Job) (err error) {
	data := job.Data()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	for _, handler := range w.handlers {
		if handler.CanProcess(data) {
			return handler.Process(ctx, job)
		}
	}

	return domain.ErrNoValidHandlerFound
}
