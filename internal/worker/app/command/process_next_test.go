package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_ProcessNext(t *testing.T) {
	processNext := func(existingJobs ...domain.Job) (func(context.Context) error, *dummyWorker) {
		store := memory.NewJobsStore(existingJobs...)
		worker := &dummyWorker{}
		return command.ProcessNext(store, store, worker), worker
	}

	t.Run("should return nil if there are no job to process", func(t *testing.T) {
		uc, worker := processNext()

		err := uc(context.Background())

		testutil.IsNil(t, err)
		testutil.IsFalse(t, worker.processedJob.HasValue())
	})

	t.Run("should process the next job", func(t *testing.T) {
		uc, worker := processNext(domain.NewJob("name", "payload"))

		err := uc(context.Background())

		testutil.IsNil(t, err)
		testutil.IsTrue(t, worker.processedJob.HasValue())
		testutil.Equals(t, "name", worker.processedJob.MustGet().Name())
		testutil.Equals(t, "payload", worker.processedJob.MustGet().Payload())

		worker.processedJob = worker.processedJob.None()

		err = uc(context.Background())
		testutil.IsNil(t, err)
		testutil.IsFalse(t, worker.processedJob.HasValue())
	})
}

type dummyWorker struct {
	processedJob monad.Maybe[domain.Job]
}

func (w *dummyWorker) Process(ctx context.Context, job domain.Job) error {
	w.processedJob = w.processedJob.WithValue(job)

	return nil
}
