package bus_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/memory"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func TestScheduler(t *testing.T) {
	t.Run("should be able to queue a job using a specific adapter", func(t *testing.T) {
		scheduler := bus.NewScheduler(memory.NewSchedulerAdapter(), must.Panic(log.NewLogger()), memory.NewBus())

		err := scheduler.Queue(context.Background(), addCommand{}, monad.None[string](), bus.JobErrPolicyRetry)

		testutil.IsNil(t, err)

		jobs, err := scheduler.GetNextPendingJobs(context.Background())

		testutil.IsNil(t, err)
		testutil.HasLength(t, jobs, 1)
	})

	t.Run("should be able to process a scheduled job and mark it as done", func(t *testing.T) {
		scheduler := bus.NewScheduler(memory.NewSchedulerAdapter(), must.Panic(log.NewLogger()), memory.NewBus())
		scheduler.Queue(context.Background(), addCommand{}, monad.None[string](), bus.JobErrPolicyRetry)
		jobs, _ := scheduler.GetNextPendingJobs(context.Background())

		// Since no handlers are attached, the job should fail and be requeued
		err := scheduler.Process(context.Background(), jobs[0])

		testutil.IsNil(t, err)

		// Since the job has the policy Retry, it should be requeued
		jobs, _ = scheduler.GetNextPendingJobs(context.Background())
		testutil.HasLength(t, jobs, 1)
	})
}
