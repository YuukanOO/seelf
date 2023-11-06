package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const retryDelay = 15 * time.Second

type (
	// VALUE OBJECTS

	JobID string

	// ENTITY

	Job struct {
		event.Emitter

		id         JobID
		name       string
		dedupeName string // Unique token to avoid multiple workers to process the same job (for example, multiple deployments for the same app and environment)
		payload    string
		errcode    monad.Maybe[string]
		queuedAt   time.Time
	}

	// RELATED SERVICES

	JobsReader interface {
		GetRunningJobs(context.Context) ([]Job, error)
		GetNextPendingJob(context.Context, []string) (Job, error)
	}

	JobsWriter interface {
		Write(context.Context, ...*Job) error
	}

	// Represents an object which can handle a specific job.
	Handler interface {
		Process(context.Context, Job) error
	}

	// EVENTS

	JobQueued struct {
		ID         JobID
		Name       string
		DedupeName string
		Payload    string
		QueuedAt   time.Time
	}

	JobDone struct {
		ID JobID
	}

	JobFailed struct {
		ID      JobID
		ErrCode string
		RetryAt time.Time
	}
)

// Creates a new job which will be processed by a worker later on.
// A dedupe name can be provided to avoid multiple workers to process the same kind of job
// at the same time, such as a deployment for the same app and environment.
// If no dedupe name is given, the job id will be used instead.
func NewJob(name, payload string, dedupeName monad.Maybe[string]) (j Job) {
	jobId := id.New[JobID]()
	j.apply(JobQueued{
		ID:         jobId,
		Name:       name,
		DedupeName: dedupeName.Get(string(jobId)),
		Payload:    payload,
		QueuedAt:   time.Now().UTC(),
	})

	return j
}

// Recreates a job from a storage scanner
func JobFrom(scanner storage.Scanner) (j Job, err error) {
	err = scanner.Scan(
		&j.id,
		&j.name,
		&j.dedupeName,
		&j.payload,
		&j.queuedAt,
		&j.errcode,
	)

	return j, err
}

// Mark the job has failed. It will be retried later on.
func (j *Job) Failed(err error) {
	j.apply(JobFailed{
		ID:      j.id,
		ErrCode: err.Error(),
		RetryAt: time.Now().UTC().Add(retryDelay),
	})
}

// Mark a job as done.
func (j *Job) Done() {
	j.apply(JobDone{
		ID: j.id,
	})
}

func (j Job) ID() JobID       { return j.id }
func (j Job) Name() string    { return j.name }
func (j Job) Payload() string { return j.payload }

func (j *Job) apply(e event.Event) {
	switch evt := e.(type) {
	case JobQueued:
		j.id = evt.ID
		j.name = evt.Name
		j.dedupeName = evt.DedupeName
		j.payload = evt.Payload
		j.queuedAt = evt.QueuedAt
	case JobFailed:
		j.errcode = j.errcode.WithValue(evt.ErrCode)
		j.queuedAt = evt.RetryAt
	}

	event.Store(j, e)
}
