package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	JobsStore interface {
		domain.JobsReader
		domain.JobsWriter
	}

	jobsStore struct {
		sqlite.Database
	}
)

func NewJobsStore(db sqlite.Database) JobsStore {
	return &jobsStore{db}
}

func (s *jobsStore) GetNextPendingJob(ctx context.Context) (domain.Job, error) {
	return builder.
		Query[domain.Job](`
			SELECT
				id
				,name
				,payload
				,queued_at
				,errcode
			FROM jobs
			WHERE queued_at <= datetime('now')
			ORDER BY queued_at
			LIMIT 1`).
		One(s, ctx, domain.JobFrom)
}

func (s *jobsStore) Write(c context.Context, jobs ...*domain.Job) error {
	return sqlite.WriteAndDispatch(s, c, jobs, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.JobQueued:
			return builder.
				Insert("jobs", builder.Values{
					"id":        evt.ID,
					"name":      evt.Name,
					"payload":   evt.Payload,
					"queued_at": evt.QueuedAt,
				}).
				Exec(s, ctx)
		case domain.JobFailed:
			return builder.
				Update("jobs", builder.Values{
					"errcode":   evt.ErrCode,
					"queued_at": evt.RetryAt,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s, ctx)
		case domain.JobDone:
			return builder.
				Command("DELETE FROM jobs WHERE id = ?", evt.ID).
				Exec(s, ctx)
		default:
			return nil
		}
	})
}
