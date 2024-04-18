package bus

import (
	"context"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var _ Scheduler = (*defaultScheduler)(nil) // Validate interface implementation

const (
	JobPolicyRetryPreserveOrder      JobPolicy = 1 << iota // Retry the job but preserve the order among the group
	JobPolicyWaitForOthersResourceID                       // Wait for other jobs on the same resource id to finish before processing
	JobPolicyCancellable                                   // The job can be cancellable by a user
)

type (
	JobPolicy uint8

	// Represents a schedulable request, one that can be queued for later dispatching.
	Schedulable interface {
		Request
		ResourceID() string // Resource id for which the task will be created
	}

	// Enable scheduled dispatching of a message.
	Scheduler interface {
		// Queue a request to be dispatched asynchronously at a later time.
		Queue(context.Context, Schedulable, ...JobOptions) error
	}

	// Job option passed down to adapter.
	CreateOptions struct {
		Group  monad.Maybe[string]
		Policy JobPolicy
	}

	JobOptions func(*CreateOptions)

	// Represents a scheduler that can be started and stopped.
	RunnableScheduler interface {
		Scheduler
		Start()
		Stop()
	}

	// Represents a request that has been queued for dispatching.
	ScheduledJob interface {
		ID() string
		Message() Request
		Policy() JobPolicy
	}

	GetJobsFilters struct {
		Page monad.Maybe[int] `form:"page"`
	}

	// Adapter used to store scheduled jobs. Could be anything from a database to a file or
	// an in-memory store.
	ScheduledJobsStore interface {
		Setup() error                                                                        // Setup the store
		Create(context.Context, Schedulable, CreateOptions) error                            // Create a new scheduled job
		Delete(context.Context, string) error                                                // Try to delete a job from the store
		GetAllJobs(context.Context, GetJobsFilters) (storage.Paginated[ScheduledJob], error) // Retrieve all jobs from the store
		GetNextPendingJobs(context.Context) ([]ScheduledJob, error)                          // Get the next pending jobs to be dispatched
		Retry(context.Context, ScheduledJob, error) error                                    // Retry the given job with the given reason
		Done(context.Context, ScheduledJob) error                                            // Mark the given job as done
	}

	defaultScheduler struct {
		bus                    Dispatcher
		pollInterval           time.Duration
		logger                 log.Logger
		store                  ScheduledJobsStore
		started                bool
		done                   []chan bool
		exitGroup              sync.WaitGroup
		groups                 []*workerGroup
		messageNameToWorkerIdx map[string]int
	}

	// Represents a worker group configuration used by a scheduler to spawn the appropriate
	// workers.
	WorkerGroup struct {
		Size     int      // Number of workers to start
		Messages []string // List of message names to handle, mandatory
	}

	workerGroup struct {
		jobs chan ScheduledJob
		size int
	}
)

// Builds up a new scheduler used to queue messages for later dispatching using the
// provided adapter.
func NewScheduler(adapter ScheduledJobsStore, log log.Logger, bus Dispatcher, pollInterval time.Duration, groups ...WorkerGroup) RunnableScheduler {
	s := &defaultScheduler{
		bus:                    bus,
		pollInterval:           pollInterval,
		logger:                 log,
		store:                  adapter,
		groups:                 make([]*workerGroup, len(groups)),
		messageNameToWorkerIdx: make(map[string]int),
	}

	for i, g := range groups {
		s.groups[i] = &workerGroup{
			jobs: make(chan ScheduledJob),
			size: g.Size,
		}

		for _, msg := range g.Messages {
			s.messageNameToWorkerIdx[msg] = i
		}
	}

	return s
}

func (s *defaultScheduler) Queue(
	ctx context.Context,
	msg Schedulable,
	options ...JobOptions,
) error {
	var opts CreateOptions

	for _, opt := range options {
		opt(&opts)
	}

	return s.store.Create(ctx, msg, opts)
}

func (s *defaultScheduler) Start() {
	if s.started {
		return
	}

	s.started = true

	s.startGroupRunners()
	s.startPolling()
}

func (s *defaultScheduler) Stop() {
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
func (s *defaultScheduler) run(fn func(<-chan bool)) {
	done := make(chan bool, 1)
	s.done = append(s.done, done)

	s.exitGroup.Add(1)
	go func(d <-chan bool) {
		defer s.exitGroup.Done()
		fn(d)
	}(done)
}

func (s *defaultScheduler) startPolling() {
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

			jobs, err := s.store.GetNextPendingJobs(context.Background())

			if err != nil {
				s.logger.Errorw("error while retrieving pending jobs",
					"error", err)
				continue
			}

			for _, job := range jobs {
				idx, handled := s.messageNameToWorkerIdx[job.Message().Name_()]

				if !handled {
					s.handleJobReturn(context.Background(), job, ErrNoHandlerRegistered)
					continue
				}

				s.groups[idx].jobs <- job
			}
		}
	})
}

func (s *defaultScheduler) handleJobReturn(ctx context.Context, job ScheduledJob, err error) {
	if err == nil {
		if err = s.store.Done(ctx, job); err != nil {
			s.logger.Errorw("error while marking job as done",
				"job", job.ID(),
				"name", job.Message().Name_(),
				"error", err)
		}
		return
	}

	s.logger.Errorw("error while processing job, it will be retried later",
		"job", job.ID(),
		"name", job.Message().Name_(),
		"error", err)

	if err = s.store.Retry(ctx, job, err); err != nil {
		s.logger.Errorw("error while retrying job",
			"job", job.ID(),
			"name", job.Message().Name_(),
			"error", err)
	}
}

func (s *defaultScheduler) startGroupRunners() {
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

// Attach the job being queued to a specific group meaning only one job of a group
// can be processed at a time.
func WithGroup(name string) JobOptions {
	return func(o *CreateOptions) {
		o.Group.Set(name)
	}
}

// Attach given policies to the job being queued. It will determine how the job
// will be handled.
func WithPolicy(policy JobPolicy) JobOptions {
	return func(o *CreateOptions) {
		o.Policy = policy
	}
}
