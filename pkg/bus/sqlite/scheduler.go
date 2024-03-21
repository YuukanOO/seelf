package sqlite

import (
	"context"
	"embed"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

var (
	//go:embed migrations/*.sql
	migrations embed.FS

	migrationsModule = sqlite.NewMigrationsModule("scheduler", "migrations", migrations)
)

type (
	job struct {
		id  string
		msg bus.Request
	}

	scheduler struct {
		db *sqlite.Database
	}
)

func (j *job) ID() string           { return j.id }
func (j *job) Message() bus.Request { return j.msg }

// Builds a new adapter persisting jobs in the given sqlite database.
// For it to work, commands must be (de)serializable using the bus.Marshallable mapper.
func NewSchedulerAdapter(db *sqlite.Database) bus.SchedulerAdapter {
	return &scheduler{db}
}

// Setup the scheduler adapter, migrate the database and reset running jobs by marking
// them as not retrieved so they will be picked up next time GetNextPendingJobs is called.
// You MUST call this method at the application startup.
func (s *scheduler) Setup() error {
	if err := s.db.Migrate(migrationsModule); err != nil {
		return err
	}

	_, err := s.db.ExecContext(context.Background(), `
		UPDATE scheduled_jobs
		SET retrieved = false
		WHERE retrieved = true`)

	return err
}

func (s *scheduler) Create(
	ctx context.Context,
	msg bus.Request,
	options bus.JobOptions,
) error {
	jobId := id.New[string]()

	return builder.
		Insert("scheduled_jobs", builder.Values{
			"id":           jobId,
			"dedupe_name":  options.DedupeName.Get(jobId), // Default to the job id if no dedupe
			"message_name": msg.Name_(),
			"message_data": msg,
			"queued_at":    time.Now().UTC(),
			"retrieved":    false,
		}).
		Exec(s.db, ctx)
}

func (s *scheduler) GetNextPendingJobs(ctx context.Context) ([]bus.ScheduledJob, error) {
	// This query will lock the database to make sure we can't retrieved the same job twice.
	return builder.
		Query[bus.ScheduledJob](`
			UPDATE scheduled_jobs
			SET retrieved = true
			WHERE id IN (SELECT id FROM (
				SELECT id, min(queued_at) FROM scheduled_jobs
				WHERE 
					retrieved = false
					AND queued_at <= DATETIME('now')
					AND dedupe_name NOT IN (SELECT DISTINCT dedupe_name FROM scheduled_jobs WHERE retrieved = true)
					GROUP BY dedupe_name
				)
			)
			RETURNING id, message_name, message_data`).
		All(s.db, ctx, jobMapper)
}

func (s *scheduler) Retry(ctx context.Context, j bus.ScheduledJob, jobErr error, preserveOrder bool) error {
	// If we don't need to preserve the order of related tasks, we simply update the job to queue it again
	// in the future.
	if !preserveOrder {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE scheduled_jobs
			SET
				errcode = ?
				,queued_at = DATETIME('now', '+15 seconds')
				,retrieved = false
			WHERE id = ?`, jobErr.Error(), j.ID(),
		); err != nil {
			return err
		}
	}

	// If instead, we want all jobs sharing the same dedupe_name to be updated all at once,
	// we should make sure to set all of them in the future by a specific amount to preserve
	// the job order.
	_, err := s.db.ExecContext(ctx, `
		UPDATE scheduled_jobs
		SET
			errcode = v.errcode
			,queued_at = v.updated_date
			,retrieved = false
		FROM (
			SELECT
				id
				,CASE WHEN id = ? THEN ? ELSE errcode END AS errcode
				,DATETIME('now', '+' || CAST(14 + 1 * ROW_NUMBER() OVER (ORDER BY queued_at) AS TEXT) || ' seconds') AS updated_date
			FROM scheduled_jobs
			WHERE dedupe_name = (SELECT dedupe_name FROM scheduled_jobs WHERE id = ?)
		) v
		WHERE scheduled_jobs.id = v.id`, j.ID(), jobErr.Error(), j.ID())

	return err
}

func (s *scheduler) Done(ctx context.Context, j bus.ScheduledJob) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM scheduled_jobs WHERE id = ?", j.ID())
	return err
}

func jobMapper(scanner storage.Scanner) (bus.ScheduledJob, error) {
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
