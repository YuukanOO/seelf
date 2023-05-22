package async

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	Worker interface {
		Start() // Start the worker (must be called in a goroutine)
		Stop()  // Wait for the current worker job to finish and returns
	}

	intervalWorker struct {
		started  bool
		logger   log.Logger
		interval time.Duration
		fn       func(context.Context) error
		done     chan bool
	}
)

// Builds a new worker which will call the given function at the given minimum rate.
func NewIntervalWorker(logger log.Logger, interval time.Duration, fn func(context.Context) error) Worker {
	return &intervalWorker{
		fn:       fn,
		logger:   logger,
		interval: interval,
		done:     make(chan bool),
	}
}

func (w *intervalWorker) Start() {
	if w.started {
		w.logger.Warn("worker already started")
		return
	}

	w.started = true

	for {
		select {
		case <-w.done:
			w.done <- true
			return

		case <-time.After(w.interval): // TODO: maybe we must compute the fetch interval so that we can substract the time taken by the last job to the fetch interval to run another loop immediately
			break
		}

		if err := w.fn(context.Background()); err != nil {
			w.logger.Errorw("worker function failed",
				"error", err)
		}
	}
}

func (w *intervalWorker) Stop() {
	if !w.started {
		return
	}

	w.done <- true
	w.logger.Info("waiting for the current job to finish")
	<-w.done
}
