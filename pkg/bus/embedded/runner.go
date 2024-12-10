package embedded

import (
	"context"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	Job interface {
		ID() string
		Command() bus.AsyncRequest
	}

	// Should be raised by the store when a job has been dismissed by a user.
	JobDismissed struct {
		bus.Notification

		ID      string
		Command bus.AsyncRequest
	}

	JobsStore interface {
		GetNextPendingJobs(context.Context) ([]Job, error)
		Failed(context.Context, Job, error) error
		Delay(context.Context, Job) error
		Done(context.Context, Job) error
	}

	Runner struct {
		dispatcher             bus.Dispatcher
		pollInterval           time.Duration
		started                bool
		store                  JobsStore
		logger                 log.Logger
		done                   chan struct{}
		exitGroup              sync.WaitGroup
		groups                 []*workerGroup
		messageNameToWorkerIdx map[string]int
	}

	workerGroup struct {
		jobs chan Job
		size int
	}

	// Represents a worker group configuration used by a scheduler to spawn the appropriate
	// workers.
	WorkerGroup struct {
		Size     int                // Number of workers to start
		Requests []bus.AsyncRequest // List of message types to handle
	}
)

func (JobDismissed) Name_() string { return "bus.event.job_dismissed" }

// In-process runner which process commands in specific worker groups using
// goroutines.
func NewRunner(
	store JobsStore,
	logger log.Logger,
	dispatcher bus.Dispatcher,
	pollInterval time.Duration,
	groups ...WorkerGroup,
) *Runner {
	s := &Runner{
		dispatcher:             dispatcher,
		pollInterval:           pollInterval,
		store:                  store,
		logger:                 logger,
		groups:                 make([]*workerGroup, len(groups)),
		messageNameToWorkerIdx: make(map[string]int),
	}

	for i, g := range groups {
		// Should always have at least one worker
		if g.Size < 1 {
			g.Size = 1
		}

		s.groups[i] = &workerGroup{
			size: g.Size,
		}

		for _, msg := range g.Requests {
			s.messageNameToWorkerIdx[msg.Name_()] = i
		}
	}

	return s
}

func (s *Runner) Start() {
	if s.started {
		return
	}

	s.started = true

	s.done = make(chan struct{}, 1)

	for _, g := range s.groups {
		g.jobs = make(chan Job)
	}

	s.startGroupRunners()
	s.startPolling()
}

func (s *Runner) Stop() {
	if !s.started {
		return
	}

	s.started = false

	s.logger.Info("waiting for current jobs to finish")

	close(s.done)

	for _, j := range s.groups {
		close(j.jobs)
	}

	s.exitGroup.Wait()
}

// Tiny helper to run a function in a goroutine and keep track of done channels.
func (s *Runner) run(fn func()) {
	s.exitGroup.Add(1)
	go func() {
		defer s.exitGroup.Done()
		fn()
	}()
}

func (s *Runner) startPolling() {
	s.run(func() {
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

			jobs, err := s.store.GetNextPendingJobs(context.Background())

			if err != nil {
				s.logger.Errorw("error while retrieving pending jobs",
					"error", err)
				continue
			}

			for _, job := range jobs {
				idx, handled := s.messageNameToWorkerIdx[job.Command().Name_()]

				if !handled {
					s.handleJobReturn(context.Background(), job, bus.AsyncResultProcessed, bus.ErrNoHandlerRegistered)
					continue
				}

				s.groups[idx].jobs <- job
			}
		}
	})
}

func (s *Runner) startGroupRunners() {
	for _, g := range s.groups {
		group := g
		for i := 0; i < group.size; i++ {
			s.run(func() {
				for job := range group.jobs {
					ctx := context.Background()
					result, err := bus.Send(s.dispatcher, ctx, job.Command())

					s.handleJobReturn(ctx, job, result, err)
				}
			})
		}
	}
}

func (s *Runner) handleJobReturn(ctx context.Context, job Job, result bus.AsyncResult, err error) {
	var storeErr error

	defer func() {
		if storeErr == nil {
			return
		}

		s.logger.Errorw("error while updating job status",
			"job", job.ID(),
			"name", job.Command().Name_(),
			"error", storeErr,
		)
	}()

	if err != nil {
		storeErr = s.store.Failed(ctx, job, err)
		s.logger.Errorw("error while processing job",
			"job", job.ID(),
			"name", job.Command().Name_(),
			"error", err,
		)
		return
	}

	if result == bus.AsyncResultDelay {
		storeErr = s.store.Delay(ctx, job)
		return
	}

	storeErr = s.store.Done(ctx, job)
}
