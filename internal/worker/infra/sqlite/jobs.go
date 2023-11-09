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

func (s *jobsStore) GetNextPendingJob(ctx context.Context, jobType []string) (domain.Job, error) {
	// This query will lock the database to make sure we can't retrieved the same job twice.
	return builder.
		Query[domain.Job](`
UPDATE jobs
SET retrieved = 1
WHERE id IN (
	SELECT id FROM jobs
	WHERE 
		retrieved = 0
		AND queued_at <= DATETIME('now')
		AND dedupe_name NOT IN (SELECT DISTINCT dedupe_name FROM jobs WHERE retrieved = 1)`).
		S(builder.Array("AND data_discriminator IN", jobType)).
		F(`ORDER BY queued_at LIMIT 1
	)
RETURNING id, dedupe_name, data_discriminator, data, queued_at, errcode`).
		One(s, ctx, domain.JobFrom)
}

func (s *jobsStore) GetRunningJobs(ctx context.Context) ([]domain.Job, error) {
	return builder.
		Query[domain.Job](`
			SELECT
				id
				,dedupe_name
				,data_discriminator
				,data
				,queued_at
				,errcode
			FROM jobs
			WHERE retrieved = 1`).
		All(s, ctx, domain.JobFrom)
}

func (s *jobsStore) Write(c context.Context, jobs ...*domain.Job) error {
	return sqlite.WriteAndDispatch(s, c, jobs, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.JobQueued:
			return builder.
				Insert("jobs", builder.Values{
					"id":                 evt.ID,
					"dedupe_name":        evt.DedupeName,
					"data_discriminator": evt.Data.Discriminator(),
					"data":               evt.Data,
					"queued_at":          evt.QueuedAt,
				}).
				Exec(s, ctx)
		case domain.JobFailed:
			return builder.
				Update("jobs", builder.Values{
					"errcode":   evt.ErrCode,
					"queued_at": evt.RetryAt,
					"retrieved": false, // The retrieved field is purely an infrastructure concerned and never manipulated by the domain.
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
