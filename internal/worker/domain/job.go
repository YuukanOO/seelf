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

		id       JobID
		name     string
		payload  string
		errcode  monad.Maybe[string]
		queuedAt time.Time
	}

	// RELATED SERVICES

	JobsReader interface {
		GetNextPendingJob(context.Context) (Job, error)
	}

	JobsWriter interface {
		Write(context.Context, ...*Job) error
	}

	Handler interface {
		Process(context.Context, Job) error
	}

	// EVENTS

	JobQueued struct {
		ID       JobID
		Name     string
		Payload  string
		QueuedAt time.Time
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
func NewJob(name, payload string) (j Job) {
	j.apply(JobQueued{
		ID:       id.New[JobID](),
		Name:     name,
		Payload:  payload,
		QueuedAt: time.Now().UTC(),
	})

	return j
}

// Recreates a job from a storage scanner
func JobFrom(scanner storage.Scanner) (j Job, err error) {
	err = scanner.Scan(
		&j.id,
		&j.name,
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
		j.payload = evt.Payload
		j.queuedAt = evt.QueuedAt
	case JobFailed:
		j.errcode = j.errcode.WithValue(evt.ErrCode)
		j.queuedAt = evt.RetryAt
	}

	event.Store(j, e)
}
