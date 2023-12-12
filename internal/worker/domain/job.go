package domain

import (
	"context"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const retryDelay = 15 * time.Second

var (
	ErrNoValidHandlerFound = errors.New("no_valid_handler_found")
	ErrInvalidPayload      = errors.New("invalid_payload")

	JobDataTypes = storage.NewDiscriminatedMapper[JobData]()
)

type (
	// VALUE OBJECTS

	JobID string

	JobData storage.Discriminated

	// ENTITY

	Job struct {
		event.Emitter

		id         JobID
		dedupeName string // Unique token to avoid multiple workers to process the same job (for example, multiple deployments for the same app and environment)
		data       JobData
		errcode    monad.Maybe[string]
		queuedAt   time.Time
	}

	// RELATED SERVICES

	JobsReader interface {
		GetRunningJobs(context.Context) ([]Job, error)
		GetNextPendingJobs(context.Context, []string) ([]Job, error)
	}

	JobsWriter interface {
		Write(context.Context, ...*Job) error
	}

	// Represents an object which can handle a specific job.
	Handler interface {
		Prepare(any) (JobData, monad.Maybe[string], error) // Try to prepare a job payload and returns the JobData needed to process it and an eventual dedupe name to use
		Process(context.Context, Job) error
	}

	// EVENTS

	JobQueued struct {
		bus.Notification

		ID         JobID
		DedupeName string
		Data       JobData
		QueuedAt   time.Time
	}

	JobDone struct {
		bus.Notification

		ID JobID
	}

	JobFailed struct {
		bus.Notification

		ID      JobID
		ErrCode string
		RetryAt time.Time
	}
)

func (JobQueued) Name_() string { return "worker.event.job_queued" }
func (JobDone) Name_() string   { return "worker.event.job_done" }
func (JobFailed) Name_() string { return "worker.event.job_failed" }

// Creates a new job which will be processed by a worker later on.
// A dedupe name can be provided to avoid multiple workers to process the same kind of job
// at the same time, such as a deployment for the same app and environment.
// If no dedupe name is given, the job id will be used instead.
func NewJob(data JobData, dedupeName monad.Maybe[string]) (j Job) {
	jobId := id.New[JobID]()
	j.apply(JobQueued{
		ID:         jobId,
		DedupeName: dedupeName.Get(string(jobId)),
		Data:       data,
		QueuedAt:   time.Now().UTC(),
	})

	return j
}

// Recreates a job from a storage scanner
func JobFrom(scanner storage.Scanner) (j Job, err error) {
	var (
		dataDiscriminator string
		dataPayload       monad.Maybe[string]
	)

	err = scanner.Scan(
		&j.id,
		&j.dedupeName,
		&dataDiscriminator,
		&dataPayload,
		&j.queuedAt,
		&j.errcode,
	)

	if err != nil {
		return j, err
	}

	j.data, err = JobDataTypes.From(dataDiscriminator, dataPayload.Get(""))

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

func (j Job) ID() JobID     { return j.id }
func (j Job) Data() JobData { return j.data }

func (j *Job) apply(e event.Event) {
	switch evt := e.(type) {
	case JobQueued:
		j.id = evt.ID
		j.dedupeName = evt.DedupeName
		j.data = evt.Data
		j.queuedAt = evt.QueuedAt
	case JobFailed:
		j.errcode = j.errcode.WithValue(evt.ErrCode)
		j.queuedAt = evt.RetryAt
	}

	event.Store(j, e)
}
