package async

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	// Generic interface for an async worker which can be started and stopped.
	Worker interface {
		Start() // Start the worker (must be called in a goroutine)
		Stop()  // Wait for the current worker job to finish and returns
	}

	// Handler func called by the async worker to process the next job with the given
	// tags (determined by worker groups).
	HandlerFunc func(context.Context, []string) error

	pool struct {
		started         bool
		exiting         bool
		logger          log.Logger
		interval        time.Duration // Interval at which each group will look for the next job to do.
		fn              HandlerFunc
		done            chan bool
		groupRunnerDone chan int
		groups          []*group
	}

	group struct {
		size      int
		available int
		tags      []string
	}
)

// Builds a new pool which will call the given handler func at the specified interval.
// The handler will be called for each group of jobs which have not reached their capacity
// and it will received the group tags as argument so that you can filter the job
// being returned.
//
// The idea behind pool groups is that some jobs need more time to complete and
// I don't want to hold back the other ones such as (in the future), sending
// emails, checking stuff, etc.
func NewPool(
	logger log.Logger,
	interval time.Duration,
	handler HandlerFunc,
	groups ...*group,
) Worker {
	return &pool{
		logger:          logger,
		interval:        interval,
		fn:              handler,
		done:            make(chan bool, 1),
		groupRunnerDone: make(chan int),
		groups:          groups,
	}
}

func (p *pool) Start() {
	if p.started {
		p.logger.Warn("worker already started")
		return
	}

	p.started = true

	var (
		delay   time.Duration
		lastRun time.Time = time.Now()
	)

	for {
		delay = p.interval - time.Since(lastRun)

		select {
		case <-p.done:
			p.exiting = true
		case groupIdx := <-p.groupRunnerDone:
			p.groups[groupIdx].available++
			continue
		case <-time.After(delay):
		}

		lastRun = time.Now()

		if p.exiting {
			if p.allGroupsDone() {
				p.done <- true
				return
			}

			continue
		}

		for idx, g := range p.groups {
			if g.available > 0 {
				g.available--
				go p.work(idx, g.tags)
			}
		}
	}
}

func (p *pool) Stop() {
	if !p.started {
		return
	}

	p.done <- true
	p.logger.Info("waiting for current jobs to finish")
	<-p.done
}

// Builds a new group for the given tags and specified number of concurrent jobs allowed.
func Group(size int, tags ...string) *group {
	return &group{size, size, tags}
}

func (p *pool) work(group int, tags []string) {
	defer func(g int) {
		p.groupRunnerDone <- g
		p.logger.Debugw("worker function done", "group", g)
	}(group)

	p.logger.Debugw("running worker function", "tags", tags, "group", group)

	if err := p.fn(context.Background(), tags); err != nil {
		p.logger.Errorw("worker function failed",
			"error", err)
	}
}

func (p *pool) allGroupsDone() bool {
	for _, g := range p.groups {
		if g.available < g.size {
			return false
		}
	}

	return true
}
