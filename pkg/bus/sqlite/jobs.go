package sqlite

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/dismiss_job"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/get_jobs"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/retry_job"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

var (
	_ embedded.Job       = (*job)(nil)
	_ embedded.JobsStore = (*JobsStore)(nil)
	_ bus.Scheduler      = (*JobsStore)(nil)
)

// Inner job representation satisfying the embedded bus.Job interface
type job struct {
	id  string
	msg bus.AsyncRequest
}

func (j *job) ID() string                { return j.id }
func (j *job) Command() bus.AsyncRequest { return j.msg }

type JobsStore struct {
	db         *sqlite.Database
	dispatcher bus.Dispatcher
}

// Builds a new adapter persisting jobs in the given sqlite database.
// For it to work, commands must be (de)serializable using the bus.Marshallable mapper.
func NewJobsStore(db *sqlite.Database, dispatcher bus.Dispatcher) *JobsStore {
	return &JobsStore{
		db:         db,
		dispatcher: dispatcher,
	}
}

func (s *JobsStore) ResetRetrievedJobs(ctx context.Context) error {
	_, err := s.db.ExecContext(context.Background(), `
		UPDATE scheduled_jobs
		SET retrieved = false
		WHERE retrieved = true`)

	return err
}

func (s *JobsStore) GetAllJobs(ctx context.Context, query get_jobs.Query) (storage.Paginated[get_jobs.Job], error) {
	return builder.
		Select[get_jobs.Job](`
			id
			,resource_id
			,[group]
			,message_name
			,message_data
			,queued_at
			,not_before
			,errcode
			,retrieved
		`).
		F("FROM scheduled_jobs ORDER BY queued_at").
		Paginate(s.db, ctx, jobQueryMapper, query.Page.Get(1), 10)
}

func (s *JobsStore) RetryJob(ctx context.Context, cmd retry_job.Command) (bus.UnitType, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE scheduled_jobs
		SET errcode = NULL
		WHERE id = ? AND errcode IS NOT NULL AND retrieved = false
	`, cmd.ID)

	if err != nil {
		return bus.Unit, err
	}

	affected, err := result.RowsAffected()

	if err != nil {
		return bus.Unit, err
	}

	if affected == 0 {
		return bus.Unit, apperr.ErrNotFound
	}

	return bus.Unit, err
}

func (s *JobsStore) DismissJob(ctx context.Context, cmd dismiss_job.Command) (bus.UnitType, error) {
	job, err := builder.
		Query[embedded.Job](`
			SELECT
				id, message_name, message_data
			FROM scheduled_jobs
			WHERE id = ? AND errcode IS NOT NULL AND retrieved = false
		`, cmd.ID).
		One(s.db, ctx, jobMapper)

	if err != nil {
		return bus.Unit, err
	}

	// Mark the job as done in the same transaction as the event dispatching
	return bus.Unit, s.db.Create(ctx, func(ctx context.Context) error {
		if err = s.Done(ctx, job); err != nil {
			return err
		}

		return s.dispatcher.Notify(ctx, embedded.JobDismissed{
			ID:      job.ID(),
			Command: job.Command(),
		})
	})
}

func (s *JobsStore) Queue(
	ctx context.Context,
	msg bus.AsyncRequest,
) error {
	now := time.Now().UTC()
	msgData, err := storage.ValueJSON(msg)

	if err != nil {
		return err
	}

	return builder.
		Insert("scheduled_jobs", builder.Values{
			"id":           id.New[string](),
			"resource_id":  msg.ResourceID(),
			"[group]":      msg.Group(),
			"message_name": msg.Name_(),
			"message_data": msgData,
			"queued_at":    now,
			"not_before":   now,
			"retrieved":    false,
		}).
		Exec(s.db, ctx)
}

func (s *JobsStore) GetNextPendingJobs(ctx context.Context) ([]embedded.Job, error) {
	// This query will lock the database to make sure we can't retrieved the same job twice.
	return builder.
		Query[embedded.Job](`
			UPDATE scheduled_jobs
			SET retrieved = true
			WHERE id IN (SELECT id FROM (
				SELECT id, MIN(not_before) FROM scheduled_jobs sj
				WHERE 
					sj.retrieved = false
					AND sj.errcode IS NULL
					AND sj.not_before <= DATETIME('now')
					AND sj.[group] NOT IN (SELECT DISTINCT [group] FROM scheduled_jobs WHERE retrieved = true)
					GROUP BY sj.[group]
				)
			)
			RETURNING id, message_name, message_data`).
		All(s.db, ctx, jobMapper)
}

func (s *JobsStore) Failed(ctx context.Context, job embedded.Job, jobErr error) error {
	_, err := s.db.ExecContext(ctx, `
			UPDATE scheduled_jobs
			SET
				errcode = ?
				,retrieved = false
			WHERE id = ?`, jobErr.Error(), job.ID(),
	)
	return err
}

func (s *JobsStore) Delay(ctx context.Context, job embedded.Job) error {
	// To preserve jobs order inside the same group, every job will be postponed by an amount relative to their ROWID
	_, err := s.db.ExecContext(ctx, `
		UPDATE scheduled_jobs
		SET
			not_before = v.updated_date
			,retrieved = false
		FROM (
			SELECT
				id
				,DATETIME('now', '+' || CAST(9 + 1 * ROW_NUMBER() OVER (ORDER BY queued_at) AS TEXT) || ' seconds') AS updated_date
			FROM scheduled_jobs
			WHERE errcode IS NULL AND [group] = ?
		) v
		WHERE scheduled_jobs.id = v.id`, job.Command().Group())
	return err
}

func (s *JobsStore) Done(ctx context.Context, job embedded.Job) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM scheduled_jobs WHERE id = ?", job.ID())
	return err
}

func jobMapper(scanner storage.Scanner) (embedded.Job, error) {
	var (
		j       job
		msgName string
		msgData string
	)

	err := scanner.Scan(
		&j.id,
		&msgName,
		&msgData,
	)

	if err != nil {
		return &j, err
	}

	j.msg, err = bus.Marshallable.From(msgName, msgData)

	return &j, err
}

func jobQueryMapper(scanner storage.Scanner) (j get_jobs.Job, err error) {
	err = scanner.Scan(
		&j.ID,
		&j.ResourceID,
		&j.Group,
		&j.MessageName,
		&j.MessageData,
		&j.QueuedAt,
		&j.NotBefore,
		&j.ErrorCode,
		&j.Retrieved,
	)

	return j, err
}
