package embedded_test

import (
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
)

func Test_Job(t *testing.T) {

	t.Run("could be created from an async request", func(t *testing.T) {
		cmd := returnCommand{}

		job := embedded.NewJob(cmd)

		assert.Equal[bus.AsyncRequest](t, cmd, job.Command())

		assert.HasNEvents(t, 1, &job)
		queued := assert.EventIs[embedded.JobQueued](t, &job, 0)
		assert.Equal[bus.AsyncRequest](t, cmd, queued.Command)
		assert.NotZero(t, queued.QueuedAt)
		assert.Equal(t, cmd.Key_(), queued.Key)
		assert.Equal(t, cmd.Key_(), job.Key())
		assert.Equal(t, cmd.Topic_(), queued.Topic)
		assert.Equal(t, embedded.JobStatusPending, queued.State.Status())
		assert.Equal(t, queued.QueuedAt, queued.State.NotBefore())
		assert.Zero(t, queued.State.ErrCode())
	})

	t.Run("could be mark as started", func(t *testing.T) {
		t.Run("should fail if already started", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})
			assert.Nil(t, job.HasStarted())

			err := job.HasStarted()

			assert.ErrorIs(t, err, embedded.ErrNotInPendingState)
			assert.HasNEvents(t, 2, &job)
		})

		t.Run("should succeed if not started before", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})

			err := job.HasStarted()

			assert.Nil(t, err)
			assert.HasNEvents(t, 2, &job)
			changed := assert.EventIs[embedded.JobStateChanged](t, &job, 1)
			assert.Equal(t, embedded.JobStatusRunning, changed.State.Status())
			assert.Zero(t, changed.State.ErrCode())
		})
	})

	t.Run("could be mark as ended", func(t *testing.T) {
		t.Run("and fail if not started before", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})

			err := job.HasEnded(nil)

			assert.ErrorIs(t, err, embedded.ErrNotInRunningState)
			assert.HasNEvents(t, 1, &job)
		})

		t.Run("without error", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})
			assert.Nil(t, job.HasStarted())

			err := job.HasEnded(nil)

			assert.Nil(t, err)
			assert.HasNEvents(t, 3, &job)
			changed := assert.EventIs[embedded.JobStateChanged](t, &job, 2)
			assert.Equal(t, embedded.JobStatusDone, changed.State.Status())
			assert.Zero(t, changed.State.ErrCode())
		})

		t.Run("with error and in this case delay the job", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})
			assert.Nil(t, job.HasStarted())
			expectedErr := errors.New("expected err")

			err := job.HasEnded(expectedErr)

			assert.Nil(t, err)
			assert.HasNEvents(t, 3, &job)

			queued := assert.EventIs[embedded.JobQueued](t, &job, 0)

			changed := assert.EventIs[embedded.JobStateChanged](t, &job, 2)
			assert.Equal(t, embedded.JobStatusPending, changed.State.Status())
			assert.True(t, changed.State.NotBefore().After(queued.QueuedAt))
			assert.Equal(t, expectedErr.Error(), changed.State.ErrCode().MustGet())
		})

		t.Run("could be started many times", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})
			assert.Nil(t, job.HasStarted())
			assert.Nil(t, job.HasEnded(errors.New("expected err")))
			assert.Nil(t, job.HasStarted())

			err := job.HasEnded(nil)

			assert.Nil(t, err)
			assert.HasNEvents(t, 5, &job)
			changed := assert.EventIs[embedded.JobStateChanged](t, &job, 4)
			assert.Equal(t, embedded.JobStatusDone, changed.State.Status())
			assert.False(t, changed.State.ErrCode().HasValue())
		})
	})

	t.Run("could be delayed", func(t *testing.T) {
		t.Run("and fail if not started before", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})

			err := job.Delay()

			assert.ErrorIs(t, err, embedded.ErrNotInRunningState)
			assert.HasNEvents(t, 1, &job)
		})

		t.Run("without issue", func(t *testing.T) {
			job := embedded.NewJob(returnCommand{})
			assert.Nil(t, job.HasStarted())

			err := job.Delay()

			assert.Nil(t, err)
			assert.HasNEvents(t, 3, &job)
			queued := assert.EventIs[embedded.JobQueued](t, &job, 0)
			delayed := assert.EventIs[embedded.JobDelayed](t, &job, 2)
			assert.Equal(t, embedded.JobStatusPending, delayed.State.Status())
			assert.NotEqual(t, queued.QueuedAt, delayed.State.NotBefore())
			assert.True(t, delayed.State.NotBefore().After(queued.QueuedAt))
			assert.Zero(t, delayed.State.ErrCode())
		})
	})
}
