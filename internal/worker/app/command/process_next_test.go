package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/async"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_ProcessNext(t *testing.T) {
	processNext := func(existingJobs ...domain.Job) (func(context.Context, async.Runner) error, *dummyHandler) {
		store := memory.NewJobsStore(existingJobs...)
		worker := &dummyHandler{}
		return command.ProcessNext(store, store, worker), worker
	}

	t.Run("should return nil if there are no job to process", func(t *testing.T) {
		uc, worker := processNext()
		runner := &dummyRunner{supportedJobs: []string{"name"}}

		err := uc(context.Background(), runner)

		testutil.IsNil(t, err)
		testutil.IsFalse(t, worker.processedJob.HasValue())
	})

	t.Run("should process the next job", func(t *testing.T) {
		uc, worker := processNext(domain.NewJob("name", "payload", monad.Value("dedupe")))
		runner := &dummyRunner{supportedJobs: []string{"name"}}

		err := uc(context.Background(), runner)

		testutil.IsNil(t, err)
		testutil.IsTrue(t, worker.processedJob.HasValue())
		testutil.Equals(t, "name", worker.processedJob.MustGet().Name())
		testutil.Equals(t, "payload", worker.processedJob.MustGet().Payload())

		testutil.DeepEquals(t, []string{"dedupe"}, runner.started)
		testutil.DeepEquals(t, []string{"dedupe"}, runner.ended)

		worker.processedJob = worker.processedJob.None()

		err = uc(context.Background(), runner)
		testutil.IsNil(t, err)
		testutil.IsFalse(t, worker.processedJob.HasValue())
	})
}

type dummyHandler struct {
	processedJob monad.Maybe[domain.Job]
}

func (w *dummyHandler) Process(ctx context.Context, job domain.Job) error {
	w.processedJob = w.processedJob.WithValue(job)

	return nil
}

type dummyRunner struct {
	supportedJobs []string
	runningJobs   []string
	started       []string
	ended         []string
}

func (r *dummyRunner) SupportedJobs() []string { return r.supportedJobs }
func (r *dummyRunner) RunningJobs() []string   { return r.runningJobs }

func (r *dummyRunner) Started(name string) {
	r.started = append(r.started, name)
	r.runningJobs = append(r.runningJobs, name)
}

func (r *dummyRunner) Ended(name string) {
	r.ended = append(r.ended, name)

	for i, n := range r.runningJobs {
		if n == name {
			r.runningJobs = append(r.runningJobs[:i], r.runningJobs[i+1:]...)
			break
		}
	}
}
