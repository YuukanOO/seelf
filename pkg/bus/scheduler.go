package bus

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
)

var _ Scheduler = (*DefaultScheduler)(nil) // Validate interface implementation

type (
	// Enable scheduled dispatching of a message.
	Scheduler interface {
		// Queue a request to be dispatched asynchronously at a later time.
		// The string parameter is the dedupe name and provide a way to avoid multiple
		// messages sharing the same dedupe name to be processed at the same time.
		Queue(context.Context, Request, ...JobOptionsBuilder) error
	}

	// Job option passed down to adapter.
	JobOptions struct {
		DedupeName monad.Maybe[string]
	}

	JobOptionsBuilder func(*JobOptions)

	// Represents a scheduler that can be started and stopped.
	RunnableScheduler interface {
		Scheduler
		Start()
		Stop()
	}

	// Represents a request that has been queued for dispatching.
	ScheduledJob interface {
		ID() string       // Unique id of the job
		Message() Request // Message to be dispatched
	}

	// Adapter used to store scheduled jobs. Could be anything from a database to a file or
	// an in-memory store.
	SchedulerAdapter interface {
		Setup() error                                               // Setup the adapter
		Create(context.Context, Request, JobOptions) error          // Create a new scheduled job
		GetNextPendingJobs(context.Context) ([]ScheduledJob, error) // Get the next pending jobs to be dispatched
		Retry(context.Context, ScheduledJob, error, bool) error     // Retry the given job with the given reason
		Done(context.Context, ScheduledJob) error                   // Mark the given job as done
	}

	DefaultScheduler struct {
		bus          Dispatcher
		pollInterval time.Duration
		logger       log.Logger
		adapter      SchedulerAdapter
		started      bool
		done         []chan bool
		exitGroup    sync.WaitGroup
		groups       []*workerGroup
	}

	// Represents a worker group configuration used by a scheduler to spawn the appropriate
	// workers.
	WorkerGroup struct {
		Size     int      // Number of workers to start
		Messages []string // List of message names to handle, leave empty to handle them all
	}

	workerGroup struct {
		jobs     chan ScheduledJob
		size     int
		messages []string
	}
)

// Builds up a new scheduler used to queue messages for later dispatching using the
// provided adapter.
func NewScheduler(adapter SchedulerAdapter, log log.Logger, bus Dispatcher, pollInterval time.Duration, groups ...WorkerGroup) RunnableScheduler {
	s := &DefaultScheduler{
		bus:          bus,
		pollInterval: pollInterval,
		logger:       log,
		adapter:      adapter,
		groups:       make([]*workerGroup, len(groups)),
	}

	for i, g := range groups {
		s.groups[i] = &workerGroup{
			jobs:     make(chan ScheduledJob),
			size:     g.Size,
			messages: g.Messages,
		}
	}

	return s
}

func (s *DefaultScheduler) Queue(
	ctx context.Context,
	msg Request,
	options ...JobOptionsBuilder,
) error {
	var opts JobOptions

	for _, opt := range options {
		opt(&opts)
	}

	return s.adapter.Create(ctx, msg, opts)
}

func (s *DefaultScheduler) Start() {
	if s.started {
		return
	}

	s.started = true

	s.startGroupRunners()
	s.startPolling()
}

func (s *DefaultScheduler) Stop() {
	if !s.started {
		return
	}

	s.logger.Info("waiting for current jobs to finish")

	for _, done := range s.done {
		done <- true
	}

	s.exitGroup.Wait()
}

// Tiny helper to run a function in a goroutine and keep track of done channels.
func (s *DefaultScheduler) run(fn func(<-chan bool)) {
	done := make(chan bool, 1)
	s.done = append(s.done, done)

	s.exitGroup.Add(1)
	go func(d <-chan bool) {
		defer s.exitGroup.Done()
		fn(d)
	}(done)
}

func (s *DefaultScheduler) startPolling() {
	s.run(func(done <-chan bool) {
		var (
			delay   time.Duration
			lastRun time.Time = time.Now()
		)

		for {
			delay = s.pollInterval - time.Since(lastRun)

			select {
			case <-done:
				return
			case <-time.After(delay):
			}

			lastRun = time.Now()

			jobs, err := s.adapter.GetNextPendingJobs(context.Background())

			if err != nil {
				s.logger.Errorw("error while retrieving pending jobs",
					"error", err)
				continue
			}

			for _, job := range jobs {
				var handled bool
				name := job.Message().Name_()

				for _, group := range s.groups {
					if len(group.messages) == 0 ||
						slices.Contains(group.messages, name) {
						handled = true
						group.jobs <- job
						break
					}
				}

				if !handled {
					s.handleJobReturn(context.Background(), job, ErrNoHandlerRegistered)
				}
			}
		}
	})
}

func (s *DefaultScheduler) handleJobReturn(ctx context.Context, job ScheduledJob, err error) {
	busErr, ok := apperr.As[Error](err)
	ignored := ok && busErr.policy == ErrorPolicyIgnore

	// No error or ignored, just mark the job as done
	if err == nil || ignored {
		if ignored {
			s.logger.Warnw("error while processing job but marked as ignored",
				"job", job.ID(),
				"name", job.Message().Name_(),
				"error", busErr.err)
		}

		if err = s.adapter.Done(ctx, job); err != nil {
			s.logger.Errorw("error while marking job as done",
				"job", job.ID(),
				"name", job.Message().Name_(),
				"error", err)
		}
		return
	}

	if !ok {
		busErr = Error{
			policy: ErrorPolicyRetry,
			err:    err,
		}
	}

	switch busErr.policy {
	case ErrorPolicyRetry:
		s.logger.Errorw("error while processing job",
			"job", job.ID(),
			"name", job.Message().Name_(),
			"policy", "retry",
			"error", busErr.err)
		err = s.adapter.Retry(ctx, job, busErr.err, false)
	case ErrorPolicyRetryPreserveOrder:
		s.logger.Errorw("error while processing job",
			"job", job.ID(),
			"name", job.Message().Name_(),
			"policy", "retry-preserve-order",
			"error", busErr.err)
		err = s.adapter.Retry(ctx, job, busErr.err, true)
	}

	if err != nil {
		s.logger.Errorw("error while updating job",
			"job", job.ID(),
			"name", job.Message().Name_(),
			"error", err)
	}
}

func (s *DefaultScheduler) startGroupRunners() {
	for _, g := range s.groups {
		group := g
		for i := 0; i < group.size; i++ {
			s.run(func(done <-chan bool) {
				for {
					select {
					case <-done:
						return
					case job := <-group.jobs:
						ctx := context.Background()
						_, err := s.bus.Send(ctx, job.Message())

						s.handleJobReturn(ctx, job, err)
					}
				}
			})
		}
	}
}

// Attach a specific dedupe name to the job being queued. This will prevent multiple
// jobs with the same dedupe name to be processed at the same time.
func WithDedupeName(name string) JobOptionsBuilder {
	return func(o *JobOptions) {
		o.DedupeName.Set(name)
	}
}
