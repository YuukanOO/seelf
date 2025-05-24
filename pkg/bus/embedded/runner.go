package embedded

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
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

func (s *runner) handleJobReturn(ctx context.Context, job Job, result bus.AsyncResult, jobErr error) {
	var (
		err error
		cmd = job.Command()
	)

	defer func() {
		if err == nil {
			return
		}

		s.orchestrator.logger.Errorw("error while updating job",
			"job", cmd.Key_(),
			"name", cmd.Name_(),
			"error", err,
		)
	}()

	if result == bus.AsyncResultDelay && jobErr == nil {
		err = job.Delay()
	} else {
		err = job.HasEnded(jobErr)

		if jobErr != nil {
			s.orchestrator.logger.Errorw("error while processing job, retrying later",
				"job", cmd.Key_(),
				"name", cmd.Name_(),
				"error", jobErr,
			)
		}
	}

	if err != nil {
		return
	}

	if err = s.orchestrator.store.Write(ctx, &job); errors.Is(err, storage.ErrConcurrencyUpdate) {
		err = s.orchestrator.store.MarkAsPending(ctx, job.Key())
	}
}
