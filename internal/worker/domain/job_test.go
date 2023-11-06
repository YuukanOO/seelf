package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Job(t *testing.T) {
	t.Run("can be created", func(t *testing.T) {
		name := "jobname"
		payload := "somepayload"

		job := domain.NewJob(name, payload, monad.None[string]())

		testutil.NotEquals(t, "", job.ID())
		testutil.Equals(t, name, job.Name())
		testutil.Equals(t, payload, job.Payload())

		testutil.HasNEvents(t, &job, 1)

		evt := testutil.EventIs[domain.JobQueued](t, &job, 0)

		testutil.Equals(t, job.ID(), evt.ID)
		testutil.Equals(t, job.Name(), evt.Name)
		testutil.Equals(t, job.Payload(), evt.Payload)
		testutil.IsFalse(t, evt.QueuedAt.IsZero())
	})

	t.Run("can be created with a dedupe name", func(t *testing.T) {
		name := "jobname"
		payload := "somepayload"

		job := domain.NewJob(name, payload, monad.None[string]())

		evt := testutil.EventIs[domain.JobQueued](t, &job, 0)
		testutil.Equals(t, string(evt.ID), evt.DedupeName)

		dedupeName := "app-environment"

		job = domain.NewJob(name, payload, monad.Value(dedupeName))

		evt = testutil.EventIs[domain.JobQueued](t, &job, 0)
		testutil.Equals(t, dedupeName, evt.DedupeName)
	})

	t.Run("can be marked as failed", func(t *testing.T) {
		err := errors.New("some error")
		job := domain.NewJob("jobname", "somepayload", monad.None[string]())

		job.Failed(err)

		testutil.HasNEvents(t, &job, 2)
		queuedEvt := testutil.EventIs[domain.JobQueued](t, &job, 0)
		firstQueuedAt := queuedEvt.QueuedAt

		failedEvt := testutil.EventIs[domain.JobFailed](t, &job, 1)
		testutil.Equals(t, job.ID(), failedEvt.ID)
		testutil.Equals(t, err.Error(), failedEvt.ErrCode)
		testutil.IsTrue(t, failedEvt.RetryAt.Sub(firstQueuedAt) >= 15*time.Second)
	})

	t.Run("can be marked as done", func(t *testing.T) {
		job := domain.NewJob("jobname", "somepayload", monad.None[string]())

		job.Done()

		testutil.HasNEvents(t, &job, 2)
		evt := testutil.EventIs[domain.JobDone](t, &job, 1)
		testutil.Equals(t, job.ID(), evt.ID)
	})
}
