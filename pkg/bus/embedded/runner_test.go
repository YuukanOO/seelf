package embedded_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Runner(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	b := embedded.NewBus()
	// Register an handler which will just return the inner cmd error to test how the scheduler behave.
	bus.Register(b, func(_ context.Context, cmd returnCommand) (bus.AsyncResult, error) {
		return cmd.result, cmd.err
	})

	t.Run("should fail the job if no handler is registered", func(t *testing.T) {
		var (
			adapter adapter
			cmd     bus.AsyncRequest = unhandledCommand{}
		)

		runner := embedded.NewRunner(&adapter, logger, b, 0, embedded.WorkerGroup{
			Requests: []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		adapter.wait()

		assert.HasLength(t, 0, adapter.done)
		assert.HasLength(t, 0, adapter.delayed)
		assert.HasLength(t, 1, adapter.failed)
		assert.Equal(t, cmd, adapter.failed[0].Command())
		assert.Equal(t, bus.ErrNoHandlerRegistered, adapter.failed[0].err)
	})

	t.Run("should fail the job if there was an error", func(t *testing.T) {
		var (
			adapter adapter
			jobErr                   = errors.New("some error")
			cmd     bus.AsyncRequest = returnCommand{err: jobErr}
		)

		runner := embedded.NewRunner(&adapter, logger, b, 0, embedded.WorkerGroup{
			Requests: []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		adapter.wait()

		assert.HasLength(t, 0, adapter.done)
		assert.HasLength(t, 0, adapter.delayed)
		assert.HasLength(t, 1, adapter.failed)
		assert.Equal(t, cmd, adapter.failed[0].Command())
		assert.Equal(t, jobErr, adapter.failed[0].err)
	})

	t.Run("should delay the job if the handler returns an AsyncResultDelay", func(t *testing.T) {
		var (
			adapter adapter
			cmd     bus.AsyncRequest = returnCommand{result: bus.AsyncResultDelay}
		)

		runner := embedded.NewRunner(&adapter, logger, b, 0, embedded.WorkerGroup{
			Requests: []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		adapter.wait()

		assert.HasLength(t, 0, adapter.done)
		assert.HasLength(t, 1, adapter.delayed)
		assert.HasLength(t, 0, adapter.failed)
		assert.Equal(t, cmd, adapter.delayed[0].Command())
		assert.Nil(t, adapter.delayed[0].err)
	})

	t.Run("should mark the job as done if there is no error", func(t *testing.T) {
		var (
			adapter adapter
			cmd     bus.AsyncRequest = returnCommand{}
		)

		runner := embedded.NewRunner(&adapter, logger, b, 0, embedded.WorkerGroup{
			Requests: []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		adapter.wait()

		assert.HasLength(t, 1, adapter.done)
		assert.HasLength(t, 0, adapter.delayed)
		assert.HasLength(t, 0, adapter.failed)
		assert.Equal(t, cmd, adapter.done[0].Command())
	})
}

type returnCommand struct {
	bus.AsyncCommand

	result bus.AsyncResult
	err    error
}

func (r returnCommand) Name_() string { return "returnCommand" }
func (r returnCommand) Group() string { return "" }

type unhandledCommand struct {
	bus.AsyncCommand
}

func (u unhandledCommand) Name_() string { return "unhandledCommand" }
func (u unhandledCommand) Group() string { return "" }

var (
	_ embedded.JobsStore = (*adapter)(nil)
	_ bus.Scheduler      = (*adapter)(nil)
)

type job struct {
	id  string
	cmd bus.AsyncRequest
	err error
}

func (j *job) ID() string                { return j.id }
func (j *job) Command() bus.AsyncRequest { return j.cmd }

type adapter struct {
	wg      sync.WaitGroup
	jobs    []*job
	failed  []*job
	delayed []*job
	done    []*job
}

func (a *adapter) Queue(ctx context.Context, requests ...bus.AsyncRequest) error {
	a.wg.Add(1)

	for _, req := range requests {
		a.jobs = append(a.jobs, &job{id: id.New[string](), cmd: req})
	}

	return nil
}

func (a *adapter) Delay(ctx context.Context, j embedded.Job) error {
	defer a.wg.Done()
	a.delayed = append(a.delayed, j.(*job))
	return nil
}

func (a *adapter) Done(ctx context.Context, j embedded.Job) error {
	defer a.wg.Done()
	a.done = append(a.done, j.(*job))
	return nil
}

func (a *adapter) Failed(ctx context.Context, j embedded.Job, jobErr error) error {
	defer a.wg.Done()
	jo := j.(*job)
	jo.err = jobErr
	a.failed = append(a.failed, jo)
	return nil
}

func (a *adapter) GetNextPendingJobs(context.Context) ([]embedded.Job, error) {
	j := make([]embedded.Job, len(a.jobs))

	for i, job := range a.jobs {
		j[i] = job
	}

	a.jobs = nil

	return j, nil
}

func (a *adapter) wait() {
	a.wg.Wait()
}
