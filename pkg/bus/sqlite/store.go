package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/flag"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
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
		id     string
		msg    bus.Request
		policy bus.JobPolicy
	}

	jobQuery struct {
		JobID       string              `json:"id"`
		ResourceID  string              `json:"resource_id"`
		Group       string              `json:"group"`
		MessageName string              `json:"message_name"`
		MessageData string              `json:"message_data"`
		QueuedAt    time.Time           `json:"queued_at"`
		NotBefore   time.Time           `json:"not_before"`
		ErrorCode   monad.Maybe[string] `json:"error_code"`
		JobPolicy   bus.JobPolicy       `json:"policy"`
		Retrieved   bool                `json:"retrieved"`
	}

	store struct {
		db *sqlite.Database
	}
)

func (j *job) ID() string            { return j.id }
func (j *job) Message() bus.Request  { return j.msg }
func (j *job) Policy() bus.JobPolicy { return j.policy }

func (j *jobQuery) ID() string            { return j.JobID }
func (j *jobQuery) Message() bus.Request  { panic("not implemented") } // Should never happen because this is a query only job
func (j *jobQuery) Policy() bus.JobPolicy { return j.JobPolicy }

// Builds a new adapter persisting jobs in the given sqlite database.
// For it to work, commands must be (de)serializable using the bus.Marshallable mapper.
func NewScheduledJobsStore(db *sqlite.Database) bus.ScheduledJobsStore {
	return &store{db}
}

// Setup the scheduler adapter, migrate the database and reset running jobs by marking
// them as not retrieved so they will be picked up next time GetNextPendingJobs is called.
// You MUST call this method at the application startup.
func (s *store) Setup() error {
	if err := s.db.Migrate(migrationsModule); err != nil {
		return err
	}

	_, err := s.db.ExecContext(context.Background(), `
		UPDATE scheduled_jobs
		SET retrieved = false
		WHERE retrieved = true`)

	return err
}

func (s *store) Create(
	ctx context.Context,
	msg bus.Schedulable,
	options bus.CreateOptions,
) error {
	jobId := id.New[string]()
	now := time.Now().UTC()
	msgValue, err := storage.ValueJSON(msg)

	if err != nil {
		return err
	}

	var (
		msgName    = msg.Name_()
		resourceId = msg.ResourceID()
	)

	// Could not use the ON CONFLICT here :'(
	if flag.IsSet(options.Policy, bus.JobPolicyMerge) {
		var existingJobId string

		if err = s.db.QueryRowContext(ctx, `
		SELECT id
		FROM scheduled_jobs
		WHERE resource_id = ? AND message_name = ? AND retrieved = false`, resourceId, msgName).
			Scan(&existingJobId); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if existingJobId != "" {
			_, err = s.db.ExecContext(ctx, `UPDATE scheduled_jobs SET message_data = ? WHERE id = ?`, msgValue, existingJobId)
			return err
		}
	}

	return builder.
		Insert("scheduled_jobs", builder.Values{
			"id":           jobId,
			"resource_id":  resourceId,
			"[group]":      options.Group.Get(jobId), // Default to the job id if no group set
			"message_name": msgName,
			"message_data": msgValue,
			"queued_at":    now,
			"not_before":   now,
			"policy":       options.Policy,
			"retrieved":    false,
		}).
		Exec(s.db, ctx)
}

func (s *store) Delete(ctx context.Context, id string) error {
	r, err := s.db.ExecContext(ctx, "DELETE FROM scheduled_jobs WHERE id = ? AND (policy & ?) != 0",
		id, bus.JobPolicyCancellable)

	if err != nil {
		return err
	}

	affected, err := r.RowsAffected()

	if err != nil {
		return err
	}

	if affected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (s *store) GetAllJobs(ctx context.Context, filters bus.GetJobsFilters) (storage.Paginated[bus.ScheduledJob], error) {
	return builder.
		Select[bus.ScheduledJob](`
			id
			,resource_id
			,[group]
			,message_name
			,message_data
			,queued_at
			,not_before
			,errcode
			,policy
			,retrieved
		`).
		F("FROM scheduled_jobs ORDER BY queued_at").
		Paginate(s.db, ctx, jobQueryMapper, filters.Page.Get(1), 10)
}

func (s *store) GetNextPendingJobs(ctx context.Context) ([]bus.ScheduledJob, error) {
	// This query will lock the database to make sure we can't retrieved the same job twice.
	return builder.
		Query[bus.ScheduledJob](`
			UPDATE scheduled_jobs
			SET retrieved = true
			WHERE id IN (SELECT id FROM (
				SELECT id, MIN(not_before) FROM scheduled_jobs sj
				WHERE 
					sj.retrieved = false
					AND sj.not_before <= DATETIME('now')
					AND sj.[group] NOT IN (SELECT DISTINCT [group] FROM scheduled_jobs WHERE retrieved = true)
					AND (sj.policy & ? = 0 OR (SELECT COUNT(resource_id) FROM scheduled_jobs WHERE resource_id = sj.resource_id) <= 1)
					GROUP BY sj.[group]
				)
			)
			RETURNING id, message_name, message_data, policy`, bus.JobPolicyWaitForOthersResourceID).
		All(s.db, ctx, jobMapper)
}

func (s *store) Retry(ctx context.Context, j bus.ScheduledJob, jobErr error) error {
	// If we don't need to preserve the order of related tasks, we simply update the job to queue it again
	// in the future.
	if !flag.IsSet(j.Policy(), bus.JobPolicyRetryPreserveOrder) {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE scheduled_jobs
			SET
				errcode = ?
				,not_before = DATETIME('now', '+15 seconds')
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
			,not_before = v.updated_date
			,retrieved = false
		FROM (
			SELECT
				id
				,CASE WHEN id = ? THEN ? ELSE errcode END AS errcode
				,DATETIME('now', '+' || CAST(14 + 1 * ROW_NUMBER() OVER (ORDER BY not_before) AS TEXT) || ' seconds') AS updated_date
			FROM scheduled_jobs
			WHERE [group] = (SELECT [group] FROM scheduled_jobs WHERE id = ?)
		) v
		WHERE scheduled_jobs.id = v.id`, j.ID(), jobErr.Error(), j.ID())

	return err
}

func (s *store) Done(ctx context.Context, j bus.ScheduledJob) error {
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
		&j.policy,
	)

	if err != nil {
		return &j, err
	}

	j.msg, err = bus.Marshallable.From(msgName, msgData)

	return &j, err
}

func jobQueryMapper(scanner storage.Scanner) (bus.ScheduledJob, error) {
	var j jobQuery

	err := scanner.Scan(
		&j.JobID,
		&j.ResourceID,
		&j.Group,
		&j.MessageName,
		&j.MessageData,
		&j.QueuedAt,
		&j.NotBefore,
		&j.ErrorCode,
		&j.JobPolicy,
		&j.Retrieved,
	)

	return &j, err
}
