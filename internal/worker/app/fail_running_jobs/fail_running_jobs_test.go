package fail_running_jobs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/fail_running_jobs"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_FailRunningJobs(t *testing.T) {
	sut := func(existingJobs ...*domain.Job) bus.RequestHandler[bool, fail_running_jobs.Command] {
		store := memory.NewJobsStore(existingJobs...)
		return fail_running_jobs.Handler(store, store)
	}

	t.Run("should reset running jobs", func(t *testing.T) {
		reason := errors.New("server_reset")
		ctx := context.Background()
		job1 := domain.NewJob(payload{}, monad.None[string]())
		job2 := domain.NewJob(payload{}, monad.None[string]())

		fail := sut(&job1, &job2)

		success, err := fail(ctx, fail_running_jobs.Command{Reason: reason})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, success)

		evt := testutil.EventIs[domain.JobFailed](t, &job1, 1)
		testutil.Equals(t, reason.Error(), evt.ErrCode)
		evt = testutil.EventIs[domain.JobFailed](t, &job2, 1)
		testutil.Equals(t, reason.Error(), evt.ErrCode)
	})
}

type payload struct{ domain.JobData }
