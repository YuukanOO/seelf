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
	processNext := func(existingJobs ...domain.Job) (func(context.Context, command.ProcessNextCommand) error, *dummyHandler) {
		store := memory.NewJobsStore(existingJobs...)
		worker := &dummyHandler{}
		return command.ProcessNext(store, store, worker), worker
	}

	t.Run("should return nil if there are no job to process", func(t *testing.T) {
		uc, worker := processNext()
		err := uc(context.Background(), command.ProcessNextCommand{
			Names: []string{"name"},
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, worker.processedJob.HasValue())
	})

	t.Run("should process the next job", func(t *testing.T) {
		var data payload

		uc, worker := processNext(domain.NewJob(data, monad.Value("dedupe")))

		err := uc(context.Background(), command.ProcessNextCommand{
			Names: []string{data.Discriminator()},
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, worker.processedJob.HasValue())
		testutil.Equals(t, data, worker.processedJob.MustGet().Data().(payload))

		worker.processedJob = worker.processedJob.None()

		err = uc(context.Background(), command.ProcessNextCommand{
			Names: []string{data.Discriminator()},
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, worker.processedJob.HasValue())
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
