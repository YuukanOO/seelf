package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/memory"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func TestScheduler(t *testing.T) {
	t.Run("should be able to queue a job", func(t *testing.T) {
		a := memory.NewSchedulerAdapter()
		err := a.Create(context.Background(), addCommand{}, monad.None[string](), bus.JobErrPolicyRetry)

		testutil.IsNil(t, err)
	})

	t.Run("should returns all job queued", func(t *testing.T) {
		var (
			a  = memory.NewSchedulerAdapter()
			c1 = addCommand{A: 1, B: 2}
			c2 = getQuery{}
		)

		a.Create(context.Background(), c1, monad.None[string](), bus.JobErrPolicyRetry)
		a.Create(context.Background(), c2, monad.None[string](), bus.JobErrPolicyRetry)

		jobs, err := a.GetNextPendingJobs(context.Background())

		testutil.IsNil(t, err)
		testutil.HasLength(t, jobs, 2)
		testutil.Equals[bus.Request](t, c1, jobs[0].Message())
		testutil.Equals[bus.Request](t, c2, jobs[1].Message())

		jobs, err = a.GetNextPendingJobs(context.Background())
		testutil.IsNil(t, err)
		testutil.HasLength(t, jobs, 0)
	})

	t.Run("should be able to mark jobs has done and satisfy the job err policy", func(t *testing.T) {
		var (
			a      = memory.NewSchedulerAdapter()
			c1     = addCommand{A: 1, B: 2}
			c2     = getQuery{}
			jobErr = errors.New("job error")
		)

		a.Create(context.Background(), c1, monad.None[string](), bus.JobErrPolicyIgnore)
		a.Create(context.Background(), c2, monad.None[string](), bus.JobErrPolicyRetry)
		jobs, _ := a.GetNextPendingJobs(context.Background())

		testutil.IsNil(t, a.Done(context.Background(), jobs[0]))
		testutil.IsNil(t, a.Retry(context.Background(), jobs[1], jobErr))

		jobs, _ = a.GetNextPendingJobs(context.Background())

		testutil.HasLength(t, jobs, 1)
		testutil.Equals[bus.Request](t, c2, jobs[0].Message())
	})
}
