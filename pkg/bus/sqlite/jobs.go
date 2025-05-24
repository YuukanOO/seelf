package sqlite

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/get_jobs"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

var (
	_ embedded.JobsStore = (*JobsStore)(nil)
	_ bus.Scheduler      = (*JobsStore)(nil)
)

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
		UPDATE [scheduler.scheduled_jobs]
		SET state_status = ?
		WHERE state_status = ?`, embedded.JobStatusPending, embedded.JobStatusRunning)

	return err
}

func (s *JobsStore) GetAllJobs(ctx context.Context, query get_jobs.Query) (storage.Paginated[get_jobs.Job], error) {
	return builder.
		Select[get_jobs.Job](`
			key
			,topic
			,message_name
			,message_data
			,queued_at
			,state_status
			,state_not_before
			,state_errcode
		`).
		F("FROM [scheduler.scheduled_jobs] ORDER BY queued_at").
		Paginate(s.db, ctx, jobQueryMapper, query.Page.Get(1), 10)
}

func (s *JobsStore) MarkAsPending(ctx context.Context, key bus.JobKey) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE [scheduler.scheduled_jobs]
		SET state_status = ?
		WHERE key = ?`, embedded.JobStatusPending, key)

	return err
}

func (s *JobsStore) Write(ctx context.Context, jobs ...*embedded.Job) error {
	return sqlite.WriteEventsBis(s.db, ctx, jobs,
		"[scheduler.scheduled_jobs]",
		func(job *embedded.Job) sqlite.Key {
			return sqlite.Key{
				"key": job.Key(),
			}
		},
		func(e event.Event, c *sqlite.Context) {
			switch evt := e.(type) {
			case embedded.JobQueued:
				c.Values["key"] = evt.Key
				c.Values["topic"] = evt.Topic
				c.Values["message_name"] = evt.Command.Name_()
				c.Values["message_data"] = must.Panic(storage.ValueJSON(evt.Command))
				c.Values["queued_at"] = evt.QueuedAt
				c.Values["state_status"] = evt.State.Status()
				c.Values["state_not_before"] = evt.State.NotBefore()
				c.Values["state_errcode"] = evt.State.ErrCode()
			case embedded.JobStateChanged:
				// No need to keep successful jobs!
				if evt.State.Status() == embedded.JobStatusDone {
					c.Delete()
					return
				}

				c.Values["state_status"] = evt.State.Status()
				c.Values["state_not_before"] = evt.State.NotBefore()
				c.Values["state_errcode"] = evt.State.ErrCode()
			case embedded.JobDelayed:
				c.Values["state_status"] = evt.State.Status()
				c.Values["state_not_before"] = evt.State.NotBefore()
				c.Values["state_errcode"] = evt.State.ErrCode()
			}
		},
		sqlite.UpdateOnConflict("key", "state_status"),
	)
}

func (s *JobsStore) Queue(
	ctx context.Context,
	requests ...bus.AsyncRequest,
) error {
	jobs := make([]*embedded.Job, len(requests))

	for i, req := range requests {
		job := embedded.NewJob(req)
		jobs[i] = &job
	}

	return s.Write(ctx, jobs...)
}

func (s *JobsStore) Cancel(ctx context.Context, keys ...bus.JobKey) error {
	if len(keys) == 0 {
		return nil
	}

	return builder.
		Command(`DELETE FROM [scheduler.scheduled_jobs]`).
		S(builder.Array("WHERE key IN", keys)).
		MustExec(s.db, ctx)
}

func (s *JobsStore) GetNextPendingJobs(ctx context.Context, messageNames ...string) ([]embedded.Job, error) {
	// This query will lock the database to make sure we can't retrieved the same job twice.
	return builder.
		Query[embedded.Job](`
			UPDATE [scheduler.scheduled_jobs]
			SET state_status = ?
			WHERE key IN (SELECT key FROM (
				SELECT key, MIN(state_not_before) FROM [scheduler.scheduled_jobs] sj
				WHERE 
					sj.state_status = ?`, embedded.JobStatusRunning, embedded.JobStatusPending).
		S(builder.Array("AND sj.message_name IN", messageNames)).
		F(`
					AND sj.state_not_before <= ?
					AND sj.topic NOT IN (SELECT DISTINCT topic FROM [scheduler.scheduled_jobs] WHERE state_status = ?)
					GROUP BY sj.topic
				)
			)
			RETURNING key, topic, message_name, message_data, queued_at, state_status, state_not_before, state_errcode, version`, time.Now().UTC(), embedded.JobStatusRunning).
		All(s.db, ctx, embedded.JobFrom)
}

func jobQueryMapper(scanner storage.Scanner) (j get_jobs.Job, err error) {
	err = scanner.Scan(
		&j.Key,
		&j.Topic,
		&j.MessageName,
		&j.MessageData,
		&j.QueuedAt,
		&j.State.Status,
		&j.State.NotBefore,
		&j.State.ErrorCode,
	)

	return j, err
}
