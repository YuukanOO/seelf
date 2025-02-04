package embedded

import (
	"context"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
)

type (

	// In-process runner which process commands in specific worker groups using
	// goroutines.
	runner struct {
		orchestrator *Orchestrator
		pollInterval time.Duration
		messages     []string
		workersCount uint8
		exitGroup    sync.WaitGroup
		jobs         chan Job
		done         chan struct{}
	}
)

func newRunner(
	orchestrator *Orchestrator,
	definition RunnerDefinition,
) *runner {
	s := &runner{
		orchestrator: orchestrator,
		pollInterval: definition.PollInterval,
		workersCount: definition.WorkersCount,
	}

	// Should always have at least one worker
	if s.workersCount < 1 {
		s.workersCount = 1
	}

	s.messages = make([]string, len(definition.Messages))

	for i, msg := range definition.Messages {
		s.messages[i] = msg.Name_()
	}

	return s
}

func (s *runner) start() {
	s.done = make(chan struct{})
	s.jobs = make(chan Job)

	s.orchestrator.logger.Debugw("starting runner",
		"poll", s.pollInterval,
		"workers", s.workersCount,
		"messages", s.messages)

	s.startWorkers()
	go s.startPolling()
}

func (s *runner) stop() {
	s.orchestrator.logger.Debug("waiting for current jobs to finish")

	close(s.done)
	close(s.jobs)

	s.exitGroup.Wait()
}

func (s *runner) startPolling() {
	s.exitGroup.Add(1)
	defer s.exitGroup.Done()

	var (
		delay   time.Duration
		lastRun time.Time = time.Now()
	)

	for {
		delay = s.pollInterval - time.Since(lastRun)

		select {
		case <-s.done:
			return
		case <-time.After(delay):
		}

		lastRun = time.Now()

		jobs, err := s.orchestrator.store.GetNextPendingJobs(context.Background(), s.messages...)

		if err != nil {
			s.orchestrator.logger.Errorw("error while retrieving pending jobs",
				"error", err)
			continue
		}

		for _, job := range jobs {
			s.jobs <- job
		}
	}
}

func (s *runner) startWorkers() {
	for range s.workersCount {
		s.exitGroup.Add(1)
		go func() {
			defer s.exitGroup.Done()

			for job := range s.jobs {
				ctx := context.Background()
				result, err := bus.Send(s.orchestrator.dispatcher, ctx, job.Command())

				s.handleJobReturn(ctx, job, result, err)
			}
		}()
	}
}

func (s *runner) handleJobReturn(ctx context.Context, job Job, result bus.AsyncResult, err error) {
	var storeErr error

	defer func() {
		if storeErr == nil {
			return
		}

		s.orchestrator.logger.Errorw("error while updating job status",
			"job", job.ID(),
			"name", job.Command().Name_(),
			"error", storeErr,
		)
	}()

	if err != nil {
		storeErr = s.orchestrator.store.Failed(ctx, job, err)
		s.orchestrator.logger.Errorw("error while processing job",
			"job", job.ID(),
			"name", job.Command().Name_(),
			"error", err,
		)
		return
	}

	if result == bus.AsyncResultDelay {
		storeErr = s.orchestrator.store.Delay(ctx, job)
		return
	}

	storeErr = s.orchestrator.store.Done(ctx, job)
}
