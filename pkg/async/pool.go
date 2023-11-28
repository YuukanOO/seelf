package async

import (
	"context"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	// Generic interface for an async worker which can be started and stopped.
	Pool interface {
		Start() // Start the pool and all its workers (must be called in a goroutine)
		Stop()  // Wait for the current pool to complete and returns
	}

	PollingFunc[T any] func(context.Context, []string) ([]T, error) // Function used to retrieve jobs to process
	NameFunc[T any]    func(T) string                               // Extract a job name
	HandlerFunc[T any] func(context.Context, T) error               // Process a single job

	pool[T any] struct {
		started                bool
		logger                 log.Logger
		interval               time.Duration // Interval at which the PollingFunc will be called
		pollingFunc            PollingFunc[T]
		nameFunc               NameFunc[T]
		done                   chan bool
		exitGroup              sync.WaitGroup // Wait group used to wait for all workers to exit when stopping
		tags                   []string       // Supported tags (passed to the polling func)
		jobs                   []chan T       // Jobs channels (one per worker group)
		tagsJobsChannelMapping map[string]int // Mapping between job name/tag and the appropriate channel index in the jobs array
		workers                []*worker[T]
	}

	group[T any] struct {
		size        int
		handlerFunc HandlerFunc[T]
		tags        []string
	}
)

// Builds a new pool which will poll for jobs at specific intervals.
// You must provide worker groups which will process specific job types retrieved
// by the polling function. This polling function will fill a channel so worker can pick
// jobs as soon as they are available.
//
// The idea behind worker groups is that some jobs need more time to complete and
// I don't want to hold back the other ones such as (in the future), sending
// emails, checking stuff, etc.
func NewPool[T any](
	logger log.Logger,
	interval time.Duration,
	pollingFunc PollingFunc[T],
	nameFunc NameFunc[T],
	groups ...*group[T],
) Pool {
	var (
		tags    []string
		jobs    []chan T
		mapping map[string]int = make(map[string]int)
		workers []*worker[T]
	)

	for idx, g := range groups {
		tags = append(tags, g.tags...)
		jobs = append(jobs, make(chan T))

		for _, t := range g.tags {
			mapping[t] = idx
		}

		for i := 0; i < g.size; i++ {
			workers = append(workers, newWorker(logger, jobs[idx], g.handlerFunc))
		}
	}

	return &pool[T]{
		logger:                 logger,
		interval:               interval,
		pollingFunc:            pollingFunc,
		nameFunc:               nameFunc,
		done:                   make(chan bool, 1),
		tags:                   tags,
		jobs:                   jobs,
		tagsJobsChannelMapping: mapping,
		workers:                workers,
	}
}

func (p *pool[T]) Start() {
	if p.started {
		p.logger.Warn("pool already started")
		return
	}

	p.started = true

	// Launch every worker registered in this pool
	for _, wo := range p.workers {
		p.exitGroup.Add(1)
		go func(w *worker[T]) {
			defer p.exitGroup.Done()
			w.Start()
		}(wo)
	}

	var (
		delay   time.Duration
		lastRun time.Time = time.Now()
	)

	for {
		delay = p.interval - time.Since(lastRun)

		select {
		case <-p.done:
			p.done <- true
			return
		case <-time.After(delay):
		}

		lastRun = time.Now()

		jobs, err := p.pollingFunc(context.Background(), p.tags)

		if err != nil {
			p.logger.Errorw("error retrieving jobs",
				"error", err)
			continue
		}

		for _, job := range jobs {
			p.jobs[p.tagsJobsChannelMapping[p.nameFunc(job)]] <- job
		}
	}
}

func (p *pool[T]) Stop() {
	if !p.started {
		return
	}

	p.done <- true
	p.logger.Info("waiting for current jobs to finish")
	<-p.done

	for _, wo := range p.workers {
		wo.Stop()
	}

	p.exitGroup.Wait()
}

// Builds a new group for the given tags and specified number of concurrent jobs allowed.
func Group[T any](size int, handlerFunc HandlerFunc[T], tags ...string) *group[T] {
	return &group[T]{size, handlerFunc, tags}
}
