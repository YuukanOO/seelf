package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/collections"
	"github.com/YuukanOO/seelf/pkg/event"
)

type (
	JobsStore interface {
		domain.JobsReader
		domain.JobsWriter
	}

	jobsStore struct {
		jobs []*jobData
	}

	jobData struct {
		id    domain.JobID
		value *domain.Job
	}
)

func NewJobsStore(existingJobs ...domain.Job) JobsStore {
	s := &jobsStore{}
	ctx := context.Background()

	s.Write(ctx, collections.ToPointers(existingJobs)...)

	return s
}

func (s *jobsStore) GetNextPendingJob(ctx context.Context) (domain.Job, error) {
	if len(s.jobs) == 0 {
		return domain.Job{}, apperr.ErrNotFound
	}

	lastJob := s.jobs[len(s.jobs)-1]

	if lastJob == nil {
		return domain.Job{}, apperr.ErrNotFound
	}

	return *lastJob.value, nil
}

func (s *jobsStore) Write(ctx context.Context, jobs ...*domain.Job) error {
	for _, job := range jobs {
		for _, e := range event.Unwrap(job) {
			switch evt := e.(type) {
			case domain.JobQueued:
				var exist bool
				for _, a := range s.jobs {
					if a.id == evt.ID {
						exist = true
						break
					}
				}

				if exist {
					continue
				}

				s.jobs = append(s.jobs, &jobData{
					id:    evt.ID,
					value: job,
				})
			case domain.JobDone:
				for idx, a := range s.jobs {
					if a.id == evt.ID {
						s.jobs = append(s.jobs[:idx], s.jobs[idx+1:]...)
						break
					}
				}
			}
		}
	}

	return nil
}
