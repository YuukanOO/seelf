package sqlite

import (
	"context"
	"embed"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
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

const retryDelay = 15 * time.Second

type (
	job struct {
		id     string
		msg    bus.Request
		policy bus.JobErrPolicy
	}

	scheduler struct {
		db *sqlite.Database
	}
)

func (j *job) ID() string               { return j.id }
func (j *job) Message() bus.Request     { return j.msg }
func (j *job) Policy() bus.JobErrPolicy { return j.policy }

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

	return builder.
		Update("scheduled_jobs", builder.Values{
			"retrieved": false,
		}).
		F("WHERE retrieved = true").
		Exec(s.db, context.Background())
}

func (s *scheduler) Create(
	ctx context.Context,
	msg bus.Request,
	dedupeName monad.Maybe[string],
	policy bus.JobErrPolicy,
) error {
	jobId := id.New[string]()
	msgData, err := bus.MarshalMessage(msg)

	if err != nil {
		return err
	}

	return builder.
		Insert("scheduled_jobs", builder.Values{
			"id":           jobId,
			"dedupe_name":  dedupeName.Get(jobId), // Default to the job id if no dedupe
			"message_name": msg.Name_(),
			"message_data": msgData,
			"queued_at":    time.Now().UTC(),
			"policy":       policy,
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
RETURNING id, message_name, message_data, policy`).
		All(s.db, ctx, jobMapper)
}

func (s *scheduler) Retry(ctx context.Context, j bus.ScheduledJob, err error) error {
	return builder.
		Update("scheduled_jobs", builder.Values{
			"errcode":   err.Error(),
			"queued_at": time.Now().Add(retryDelay).UTC(),
			"retrieved": false,
		}).
		F("WHERE id = ?", j.ID()).
		Exec(s.db, ctx)
}

func (s *scheduler) Done(ctx context.Context, j bus.ScheduledJob) error {
	return builder.
		Command("DELETE FROM scheduled_jobs WHERE id = ?", j.ID()).
		Exec(s.db, ctx)
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
