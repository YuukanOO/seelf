package infra

import (
	"context"
	"fmt"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	worker struct {
		logger   log.Logger
		handlers map[string]JobsHandler
	}

	JobsHandler interface {
		domain.Handler
		JobName() string // Retrieve the name of the job handled by this handler.
	}
)

func NewHandlerFacade(logger log.Logger, handlers ...JobsHandler) domain.Handler {
	handlersMap := map[string]JobsHandler{}

	for _, handler := range handlers {
		name := handler.JobName()

		// Should never happened, but let's make it clear
		if _, exists := handlersMap[name]; exists {
			panic("duplicate job handler for " + name)
		}

		handlersMap[name] = handler
	}

	return &worker{logger, handlersMap}
}

func (w *worker) Process(ctx context.Context, job domain.Job) (err error) {
	handler, found := w.handlers[job.Name()]

	if !found {
		w.logger.Errorw("could not find a job handler for",
			"name", job.Name())
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	err = handler.Process(ctx, job)
	return
}
