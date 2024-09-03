package bus_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/memory"
	"github.com/YuukanOO/seelf/pkg/flag"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/storage"
)

func TestScheduler(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	b := memory.NewBus()
	// Register an handler which will just return the inner cmd error to test how the scheduler behave.
	bus.Register(b, func(_ context.Context, cmd returnCommand) (bus.UnitType, error) {
		return bus.Unit, cmd.err
	})

	t.Run("should queue and handle the job return appropriately", func(t *testing.T) {
		adapter := &adapter{}
		scheduler := bus.NewScheduler(adapter, logger, b, 0, bus.WorkerGroup{
			Size:     4,
			Messages: []string{returnCommand{}.Name_()},
		})

		scheduler.Start()
		defer scheduler.Stop()

		innerErr := errors.New("some error")

		withoutErr := returnCommand{}
		withUnwrapedErr := returnCommand{err: innerErr}
		withPreservedOrderErr := returnCommand{err: innerErr}

		assert.Nil(t, scheduler.Queue(context.Background(), withoutErr))
		assert.Nil(t, scheduler.Queue(context.Background(), withUnwrapedErr))
		assert.Nil(t, scheduler.Queue(context.Background(), withPreservedOrderErr, bus.WithPolicy(bus.JobPolicyRetryPreserveOrder)))
		assert.Nil(t, scheduler.Queue(context.Background(), addCommand{}))

		adapter.wait()

		assert.HasLength(t, 1, adapter.done)
		slices.SortFunc(adapter.done, func(a, b *job) int {
			return a.id - b.id
		})

		assert.Equal(t, 0, adapter.done[0].id)

		assert.HasLength(t, 3, adapter.retried)
		slices.SortFunc(adapter.retried, func(a, b *job) int {
			return a.id - b.id
		})

		assert.Equal(t, 1, adapter.retried[0].id)
		assert.ErrorIs(t, innerErr, adapter.retried[0].err)
		assert.False(t, adapter.retried[0].preserveOrder)

		assert.Equal(t, 2, adapter.retried[1].id)
		assert.ErrorIs(t, innerErr, adapter.retried[1].err)
		assert.True(t, adapter.retried[1].preserveOrder)

		assert.Equal(t, 3, adapter.retried[2].id)
		assert.ErrorIs(t, bus.ErrNoHandlerRegistered, adapter.retried[2].err)
	})
}

var (
	_ bus.ScheduledJob       = (*job)(nil)
	_ bus.ScheduledJobsStore = (*adapter)(nil)
	_ bus.Request            = (*returnCommand)(nil)
)

type (
	job struct {
		id            int
		msg           bus.Request
		policy        bus.JobPolicy
		err           error
		preserveOrder bool
	}

	adapter struct {
		wg      sync.WaitGroup
		jobs    []*job
		done    []*job
		retried []*job
	}

	returnCommand struct {
		bus.Command[bus.UnitType]

		err error
	}
)

func (r returnCommand) Name_() string      { return "returnCommand" }
func (r returnCommand) ResourceID() string { return "" }

func (j *job) ID() string            { return strconv.Itoa(j.id) }
func (j *job) Message() bus.Request  { return j.msg }
func (j *job) Policy() bus.JobPolicy { return j.policy }

func (a *adapter) Setup() error { return nil }

func (a *adapter) GetAllJobs(context.Context, bus.GetJobsFilters) (storage.Paginated[bus.ScheduledJob], error) {
	return storage.Paginated[bus.ScheduledJob]{}, nil
}

func (a *adapter) Create(_ context.Context, msg bus.Schedulable, opts bus.CreateOptions) error {
	a.wg.Add(1)
	a.jobs = append(a.jobs, &job{id: len(a.jobs), msg: msg, policy: opts.Policy})
	return nil
}

func (a *adapter) Delete(context.Context, string) error { return nil }

func (a *adapter) wait() {
	a.wg.Wait()
}

func (a *adapter) GetNextPendingJobs(context.Context) ([]bus.ScheduledJob, error) {
	j := make([]bus.ScheduledJob, len(a.jobs))

	for i, job := range a.jobs {
		j[i] = job
	}

	a.jobs = nil

	return j, nil
}

func (a *adapter) Retry(_ context.Context, j bus.ScheduledJob, jobErr error) error {
	defer a.wg.Done()
	jo := j.(*job)
	jo.err = jobErr
	jo.preserveOrder = flag.IsSet(j.Policy(), bus.JobPolicyRetryPreserveOrder)

	a.retried = append(a.retried, jo)
	return nil

}

func (a *adapter) Done(_ context.Context, j bus.ScheduledJob) error {
	defer a.wg.Done()
	a.done = append(a.done, j.(*job))
	return nil
}
