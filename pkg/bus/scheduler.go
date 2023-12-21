package bus

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
)

const (
	JobErrPolicyRetry  JobErrPolicy = iota // Retry the job if it fails
	JobErrPolicyIgnore                     // Mark the job as done even if an error is returned
)

var _ Scheduler = (*DefaultScheduler)(nil) // Validate interface implementation

type (
	JobErrPolicy uint8

	// Enable scheduled dispatching of a message.
	Scheduler interface {
		// Queue a request to be dispatched asynchronously at a later time.
		// The string parameter is the dedupe name and provide a way to avoid multiple
		// messages sharing the same dedupe name to be processed at the same time.
		Queue(context.Context, Request, monad.Maybe[string], JobErrPolicy) error
	}

	// Represents a request that has been queued for dispatching.
	ScheduledJob interface {
		ID() string           // Unique id of the job
		Message() Request     // Message to be dispatched
		Policy() JobErrPolicy // What to do when the dispatch has failed
	}

	// Adapter used to store scheduled jobs. Could be anything from a database to a file or
	// an in-memory store.
	SchedulerAdapter interface {
		Setup() error                                                             // Setup the adapter
		Create(context.Context, Request, monad.Maybe[string], JobErrPolicy) error // Create a new scheduled job
		GetNextPendingJobs(context.Context) ([]ScheduledJob, error)               // Get the next pending jobs to be dispatched
		Retry(context.Context, ScheduledJob, error) error                         // Retry the given job with the given reason
		Done(context.Context, ScheduledJob) error                                 // Mark the given job as done
	}

	DefaultScheduler struct {
		bus     Dispatcher
		logger  log.Logger
		adapter SchedulerAdapter
	}
)

// Builds up a new scheduler used to queue messages for later dispatching using the
// provided adapter.
func NewScheduler(adapter SchedulerAdapter, log log.Logger, bus Dispatcher) *DefaultScheduler {
	return &DefaultScheduler{
		bus:     bus,
		logger:  log,
		adapter: adapter,
	}
}

func (s *DefaultScheduler) Queue(
	ctx context.Context,
	msg Request,
	dedupeName monad.Maybe[string],
	policy JobErrPolicy,
) error {
	return s.adapter.Create(ctx, msg, dedupeName, policy)
}

func (s *DefaultScheduler) GetNextPendingJobs(ctx context.Context) ([]ScheduledJob, error) {
	return s.adapter.GetNextPendingJobs(ctx)
}

func (s *DefaultScheduler) Process(ctx context.Context, job ScheduledJob) error {
	_, err := s.bus.Send(ctx, job.Message())

	if err != nil {
		s.logger.Errorw("error when processing scheduled job",
			"job", job.ID(),
			"error", err)

		if job.Policy() == JobErrPolicyRetry {
			return s.adapter.Retry(ctx, job, err)
		}
	}

	return s.adapter.Done(ctx, job)
}
