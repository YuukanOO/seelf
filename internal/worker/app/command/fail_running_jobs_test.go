package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_FailRunningJobs(t *testing.T) {
	sut := func(existingJobs ...domain.Job) (func(context.Context, error) error, memory.JobsStore) {
		store := memory.NewJobsStore(existingJobs...)
		return command.FailRunningJobs(store, store), store
	}

	t.Run("should reset running jobs", func(t *testing.T) {
		reason := errors.New("server_reset")
		ctx := context.Background()
		job1 := domain.NewJob("1", "", monad.None[string]())
		job2 := domain.NewJob("2", "", monad.None[string]())

		fail, store := sut(job1, job2)

		err := fail(ctx, reason)

		testutil.IsNil(t, err)
		job1, err = store.GetByID(ctx, job1.ID())
		testutil.IsNil(t, err)
		job2, err = store.GetByID(ctx, job2.ID())
		testutil.IsNil(t, err)

		evt := testutil.EventIs[domain.JobFailed](t, &job1, 1)
		testutil.Equals(t, reason.Error(), evt.ErrCode)
		evt = testutil.EventIs[domain.JobFailed](t, &job2, 1)
		testutil.Equals(t, reason.Error(), evt.ErrCode)
	})
}
