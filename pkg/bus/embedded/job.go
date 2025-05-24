package embedded

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const (
	delayInterval   = 10 * time.Second
	errorRetryDelay = 15 * time.Second
)

type (
	// Represents a scheduled job around an async request. I found it easier to
	// manage it as an entity like every other domain aggregates.
	Job struct {
		event.Emitter

		key      bus.JobKey
		topic    string
		command  bus.AsyncRequest
		state    JobState
		queuedAt time.Time
	}

	JobsStore interface {
		GetNextPendingJobs(context.Context, ...string) ([]Job, error)
		Write(context.Context, ...*Job) error
		MarkAsPending(context.Context, bus.JobKey) error
	}

	JobQueued struct {
		bus.Notification

		Key      bus.JobKey
		Topic    string
		Command  bus.AsyncRequest
		State    JobState
		QueuedAt time.Time
	}

	JobStateChanged struct {
		bus.Notification

		Key   bus.JobKey
		State JobState
	}

	JobDelayed struct {
		bus.Notification

		Key   bus.JobKey
		State JobState
	}
)

func (JobQueued) Name_() string       { return "embedded.event.job_queued" }
func (JobStateChanged) Name_() string { return "embedded.event.job_state_changed" }
func (JobDelayed) Name_() string      { return "embedded.event.job_delayed" }

func NewJob(cmd bus.AsyncRequest) (j Job) {
	now := time.Now().UTC()
	state := JobState{
		notBefore: now,
	}

	j.apply(JobQueued{
		Key:      cmd.Key_(),
		Topic:    cmd.Topic_(),
		Command:  cmd,
		QueuedAt: now,
		State:    state,
	})
	return j
}

func JobFrom(scanner storage.Scanner) (j Job, err error) {
	var (
		version     event.Version
		messageName string
		messageData string
	)

	if err = scanner.Scan(
		&j.key,
		&j.topic,
		&messageName,
		&messageData,
		&j.queuedAt,
		&j.state.status,
		&j.state.notBefore,
		&j.state.errCode,
		&version,
	); err != nil {
		return j, err
	}

	event.Hydrate(&j, version)

	j.command, err = bus.Marshallable.From(messageName, messageData)

	return j, err
}

// Mark a job as started. Using an SQL store would probably never call it directly because when
// a job is retrieved, it will be updated in the same transaction.
func (j *Job) HasStarted() error {
	if err := j.state.started(); err != nil {
		return err
	}

	j.stateChanged()

	return nil
}

func (j *Job) HasEnded(err error) error {
	if err := j.state.ended(err); err != nil {
		return err
	}

	j.stateChanged()

	return nil
}

func (j *Job) Delay() error {
	if err := j.state.delay(time.Now().UTC().Add(delayInterval)); err != nil {
		return err
	}

	j.apply(JobDelayed{
		Key:   j.key,
		State: j.state,
	})

	return nil
}

func (j *Job) Key() bus.JobKey           { return j.key }
func (j *Job) Command() bus.AsyncRequest { return j.command }

func (j *Job) stateChanged() {
	j.apply(JobStateChanged{
		Key:   j.key,
		State: j.state,
	})
}

func (j *Job) apply(e event.Event) {
	switch evt := e.(type) {
	case JobQueued:
		j.key = evt.Key
		j.topic = evt.Topic
		j.command = evt.Command
		j.state = evt.State
		j.queuedAt = evt.QueuedAt

	case JobStateChanged:
		j.state = evt.State

	case JobDelayed:
		j.state = evt.State
	}

	event.Store(j, e)
}

var (
	ErrNotInPendingState = apperr.New("not_in_pending_state")
	ErrNotInRunningState = apperr.New("not_in_running_state")
)

const (
	JobStatusPending JobStatus = iota
	JobStatusRunning
	JobStatusDone
)

type (
	JobStatus uint8

	JobState struct {
		status    JobStatus
		notBefore time.Time
		errCode   monad.Maybe[string]
	}
)

func (s JobState) Status() JobStatus            { return s.status }
func (s JobState) NotBefore() time.Time         { return s.notBefore }
func (s JobState) ErrCode() monad.Maybe[string] { return s.errCode }

func (s *JobState) started() error {
	if s.status != JobStatusPending {
		return ErrNotInPendingState
	}

	s.status = JobStatusRunning

	return nil
}

func (s *JobState) ended(err error) error {
	if s.status != JobStatusRunning {
		return ErrNotInRunningState
	}

	if err == nil {
		s.status = JobStatusDone
		s.errCode.Unset()
		return nil
	}

	s.status = JobStatusPending
	s.notBefore = time.Now().UTC().Add(errorRetryDelay)
	s.errCode.Set(err.Error())

	return nil
}

func (s *JobState) delay(till time.Time) error {
	if s.status != JobStatusRunning {
		return ErrNotInRunningState
	}

	s.status = JobStatusPending
	s.notBefore = till

	return nil
}
