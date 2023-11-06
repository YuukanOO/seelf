package jobs

import (
	"context"
	"fmt"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	facade struct {
		logger   log.Logger
		handlers map[string]Handler
	}

	Handler interface {
		domain.Handler
		JobName() string // Retrieve the name of the job handled by this handler.
	}
)

func NewFacade(logger log.Logger, handlers ...Handler) domain.Handler {
	handlersMap := make(map[string]Handler, len(handlers))

	for _, handler := range handlers {
		name := handler.JobName()

		// Should never happened, but let's make it clear
		if _, exists := handlersMap[name]; exists {
			panic("duplicate job handler for " + name)
		}

		handlersMap[name] = handler
	}

	return &facade{logger, handlersMap}
}

func (w *facade) Process(ctx context.Context, job domain.Job) (err error) {
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
