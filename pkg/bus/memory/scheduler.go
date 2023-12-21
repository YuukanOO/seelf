package memory

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
)

type (
	job struct {
		id     string
		msg    bus.Request
		policy bus.JobErrPolicy
	}

	scheduler struct {
		jobs []*job
	}
)

func (j *job) ID() string               { return j.id }
func (j *job) Message() bus.Request     { return j.msg }
func (j *job) Policy() bus.JobErrPolicy { return j.policy }

// Builds a new inmemory scheduler adapter which will store all scheduled jobs in memory.
// It will not make use of dedupe names and should be used only for testing purposes.
func NewSchedulerAdapter() bus.SchedulerAdapter {
	return &scheduler{}
}

func (s *scheduler) Setup() error {
	return nil
}

func (s *scheduler) Create(
	_ context.Context,
	msg bus.Request,
	dedupeName monad.Maybe[string],
	policy bus.JobErrPolicy,
) error {
	s.jobs = append(s.jobs, &job{
		id:     id.New[string](),
		msg:    msg,
		policy: policy,
	})
	return nil
}

func (s *scheduler) GetNextPendingJobs(_ context.Context) ([]bus.ScheduledJob, error) {
	jobs := make([]bus.ScheduledJob, len(s.jobs))
	for i, j := range s.jobs {
		jobs[i] = j
	}
	s.jobs = nil // Reset jobs queue
	return jobs, nil
}

func (s *scheduler) Retry(_ context.Context, j bus.ScheduledJob, err error) error {
	s.jobs = append(s.jobs, j.(*job))

	return nil
}

func (s *scheduler) Done(_ context.Context, j bus.ScheduledJob) error {
	return nil
}
