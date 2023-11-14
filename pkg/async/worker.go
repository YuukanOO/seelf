package async

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/log"
)

type worker[T any] struct {
	jobs        <-chan T
	done        chan bool
	logger      log.Logger
	handlerFunc HandlerFunc[T]
}

func newWorker[T any](logger log.Logger, jobs <-chan T, handlerFunc HandlerFunc[T]) *worker[T] {
	return &worker[T]{
		logger:      logger,
		jobs:        jobs,
		done:        make(chan bool, 1),
		handlerFunc: handlerFunc,
	}
}

func (w *worker[T]) Start() {
	for {
		select {
		case job := <-w.jobs:
			if err := w.handlerFunc(context.Background(), job); err != nil {
				w.logger.Errorw("error processing job",
					"error", err,
					"job", job)
			}
		case <-w.done:
			return
		}
	}
}

func (w *worker[T]) Stop() {
	w.done <- true
}
