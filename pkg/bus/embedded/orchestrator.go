package embedded

import (
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	// Manage multiple embedded runners.
	Orchestrator struct {
		started    bool
		store      JobsStore
		dispatcher bus.Dispatcher
		logger     log.Logger
		runners    []*runner
	}

	RunnerDefinition struct {
		PollInterval time.Duration      // Interval at which messages are polled from the store
		WorkersCount uint8              // Number of workers to process incoming messages
		Messages     []bus.AsyncRequest // List of messages types processed by this runner
	}
)

func NewOrchestrator(
	store JobsStore,
	dispatcher bus.Dispatcher,
	logger log.Logger,
	definitions ...RunnerDefinition,
) *Orchestrator {
	o := &Orchestrator{
		store:      store,
		dispatcher: dispatcher,
		logger:     logger,
		runners:    make([]*runner, len(definitions)),
	}

	for i, def := range definitions {
		o.runners[i] = newRunner(o, def)
	}

	return o
}

func (o *Orchestrator) Start() {
	if o.started {
		return
	}

	o.started = true

	o.logger.Debug("starting background services")

	for _, r := range o.runners {
		go r.start()
	}
}

func (o *Orchestrator) Stop() {
	if !o.started {
		return
	}

	var wg sync.WaitGroup

	o.logger.Info("waiting for background services to complete")

	for _, r := range o.runners {
		wg.Add(1)
		go func(r *runner) {
			defer wg.Done()
			r.stop()
		}(r)
	}

	wg.Wait()
	o.started = false
}
