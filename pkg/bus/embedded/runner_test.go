package embedded_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Runner(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	b := embedded.NewBus()
	// Register an handler which will just return the inner cmd error to test how runners behave.
	bus.Register(b, func(_ context.Context, cmd returnCommand) (bus.AsyncResult, error) {
		return cmd.result, cmd.err
	})

	t.Run("should handle all jobs if no message types are given", func(t *testing.T) {
		var (
			adapter adapter
			cmd     bus.AsyncRequest = unhandledCommand{}
		)

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		runner := embedded.NewOrchestrator(&adapter, b, logger, embedded.RunnerDefinition{
			PollInterval: 0,
			WorkersCount: 1,
		})

		runner.Start()
		defer runner.Stop()

		adapter.wait()
		assert.HasLength(t, 0, adapter.done)
		assert.HasLength(t, 0, adapter.delayed)
		assert.HasLength(t, 1, adapter.failed)
		assert.Equal(t, cmd, adapter.failed[0].Command())
		changed := assert.EventIs[embedded.JobStateChanged](t, &adapter.failed[0], 2)
		assert.Equal(t, bus.ErrNoHandlerRegistered.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should fail the job if there was an error", func(t *testing.T) {
		var (
			adapter adapter
			jobErr                   = errors.New("some error")
			cmd     bus.AsyncRequest = returnCommand{err: jobErr}
		)

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		runner := embedded.NewOrchestrator(&adapter, b, logger, embedded.RunnerDefinition{
			PollInterval: 0,
			WorkersCount: 1,
			Messages:     []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

		adapter.wait()

		assert.HasLength(t, 0, adapter.done)
		assert.HasLength(t, 0, adapter.delayed)
		assert.HasLength(t, 1, adapter.failed)
		assert.Equal(t, cmd, adapter.failed[0].Command())
		changed := assert.EventIs[embedded.JobStateChanged](t, &adapter.failed[0], 2)
		assert.Equal(t, jobErr.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should delay the job if the handler returns an AsyncResultDelay", func(t *testing.T) {
		var (
			adapter adapter
			cmd     bus.AsyncRequest = returnCommand{result: bus.AsyncResultDelay}
		)

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		runner := embedded.NewOrchestrator(&adapter, b, logger, embedded.RunnerDefinition{
			PollInterval: 0,
			WorkersCount: 1,
			Messages:     []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

		adapter.wait()
		assert.HasLength(t, 0, adapter.done)
		assert.HasLength(t, 1, adapter.delayed)
		assert.HasLength(t, 0, adapter.failed)
		assert.Equal(t, cmd, adapter.delayed[0].Command())
		delayed := assert.EventIs[embedded.JobDelayed](t, &adapter.delayed[0], 2)
		assert.False(t, delayed.State.ErrCode().HasValue())
	})

	t.Run("should mark the job as done if there is no error", func(t *testing.T) {
		var (
			adapter adapter
			cmd     bus.AsyncRequest = returnCommand{}
		)

		assert.Nil(t, adapter.Queue(context.Background(), cmd))

		runner := embedded.NewOrchestrator(&adapter, b, logger, embedded.RunnerDefinition{
			PollInterval: 0,
			WorkersCount: 1,
			Messages:     []bus.AsyncRequest{returnCommand{}},
		})

		runner.Start()
		defer runner.Stop()

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

func (r returnCommand) Name_() string    { return "returnCommand" }
func (r returnCommand) Key_() bus.JobKey { return "returnKey" }
func (r returnCommand) Topic_() string   { return "" }

type unhandledCommand struct {
	bus.AsyncCommand
}

func (u unhandledCommand) Name_() string    { return "unhandledCommand" }
func (u unhandledCommand) Key_() bus.JobKey { return "unhandledKey" }
func (u unhandledCommand) Topic_() string   { return "" }

var (
	_ embedded.JobsStore = (*adapter)(nil)
	_ bus.Scheduler      = (*adapter)(nil)
)

type adapter struct {
	wg              sync.WaitGroup
	jobs            []embedded.Job
	failed          []embedded.Job
	delayed         []embedded.Job
	done            []embedded.Job
	markedAsPending []bus.JobKey
}

func (a *adapter) Queue(ctx context.Context, requests ...bus.AsyncRequest) error {
	a.wg.Add(1)

	for _, req := range requests {
		a.jobs = append(a.jobs, embedded.NewJob(req))
	}

	return nil
}

func (a *adapter) Cancel(ctx context.Context, keys ...bus.JobKey) error {
	return nil
}

func (a *adapter) Write(_ context.Context, jobs ...*embedded.Job) error {
	for _, j := range jobs {
		_, events := event.Unwrap(j)

		// Take the last changed event
		e := events[len(events)-1]

		switch evt := e.(type) {
		case embedded.JobDelayed:
			a.delayed = append(a.delayed, *j)
		case embedded.JobStateChanged:
			if evt.State.Status() == embedded.JobStatusDone {
				a.done = append(a.done, *j)
			} else if evt.State.ErrCode().HasValue() {
				a.failed = append(a.failed, *j)
			}
		}

		a.wg.Done()
	}

	return nil
}

func (a *adapter) MarkAsPending(_ context.Context, key bus.JobKey) error {
	a.markedAsPending = append(a.markedAsPending, key)
	return nil
}

func (a *adapter) GetNextPendingJobs(_ context.Context, messageNames ...string) ([]embedded.Job, error) {
	var j []embedded.Job

	for i := 0; i < len(a.jobs); i++ {
		if len(messageNames) > 0 && !slices.Contains(messageNames, a.jobs[i].Command().Name_()) {
			continue
		}

		job := a.jobs[i]
		_ = job.HasStarted()

		j = append(j, job)
		a.jobs = append(a.jobs[:i], a.jobs[i+1:]...)
		i--
	}

	return j, nil
}

func (a *adapter) wait() {
	a.wg.Wait()
}
