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
		GetByID(context.Context, domain.JobID) (domain.Job, error) // Defined here because not used in the domain
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

func (s *jobsStore) GetNextPendingJob(ctx context.Context, names []string) (domain.Job, error) {
	for _, job := range s.jobs {
		for _, name := range names {
			if job.value.Name() == name {
				return *job.value, nil
			}
		}
	}

	return domain.Job{}, apperr.ErrNotFound
}

func (s *jobsStore) GetByID(ctx context.Context, id domain.JobID) (domain.Job, error) {
	for _, job := range s.jobs {
		if job.id == id {
			return *job.value, nil
		}
	}

	return domain.Job{}, apperr.ErrNotFound
}

func (s *jobsStore) GetRunningJobs(ctx context.Context) ([]domain.Job, error) {
	// For this implementation, just returns all the jobs
	result := make([]domain.Job, len(s.jobs))

	for idx, job := range s.jobs {
		result[idx] = *job.value
	}

	return result, nil
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
			default:
				for _, d := range s.jobs {
					if d.id == job.ID() {
						d.value = job
						break
					}
				}
			}
		}
	}

	return nil
}
