package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Process(t *testing.T) {
	process := func(existingJobs ...domain.Job) (func(context.Context, domain.Job) error, memory.JobsStore, *dummyHandler) {
		store := memory.NewJobsStore(existingJobs...)
		worker := &dummyHandler{}
		return command.Process(store, worker), store, worker
	}

	t.Run("should process the given job", func(t *testing.T) {
		job := domain.NewJob(payload{}, monad.Value("dedupe"))

		uc, store, worker := process(job)

		err := uc(context.Background(), job)

		testutil.IsNil(t, err)

		_, err = store.GetByID(context.Background(), job.ID())

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.IsTrue(t, worker.processedJob.HasValue())
		testutil.Equals(t, job.ID(), worker.processedJob.MustGet().ID())
	})
}

type dummyHandler struct {
	processedJob monad.Maybe[domain.Job]
	err          error
}

func (w *dummyHandler) Process(ctx context.Context, job domain.Job) error {
	if w.err != nil {
		return w.err
	}

	w.processedJob = w.processedJob.WithValue(job)

	return nil
}

func (w *dummyHandler) Prepare(data any) (domain.JobData, monad.Maybe[string], error) {
	if w.err != nil {
		return nil, monad.None[string](), w.err
	}

	return payload{data}, monad.None[string](), nil
}

type payload struct {
	data any
}

func (p payload) Discriminator() string { return "test" }
