package async

import (
	"context"
	"sync"
	"time"

	"github.com/YuukanOO/seelf/pkg/log"
)

type (
	// Generic interface for an async worker which can be started and stopped.
	Worker interface {
		Start() // Start the worker (must be called in a goroutine)
		Stop()  // Wait for the current worker job to finish and returns
	}

	// Represents a runner which can process jobs.
	Runner interface {
		Started(string)          // Mark a job as started, you can pass whatever you want and it will be returned by RunningJobs.
		Ended(string)            // Mark a job as ended.
		SupportedJobs() []string // Retrieve the list of jobs supported by this runner.
		RunningJobs() []string   // Retrieve the list of jobs currently running (started with `.Started`).
	}

	pool struct {
		mu            sync.RWMutex // Lock used to update the currentJobs array. FIXME: maybe we can avoid it somehow?
		started       bool
		exiting       bool
		logger        log.Logger
		interval      time.Duration // Minimum interval to wait before calling the handler func.
		size          int           // Size of the pool (how many jobs can be run in parallel).
		supportedJobs []string
		currentJobs   []string
		fn            func(context.Context, Runner) error
		done          chan bool
	}
)

// Builds a new pool for the given job names which will call the handler func
// at a given rate if the pool has not reached its capacity.
// The function will be called in a goroutine and will be given a Runner to mark
// jobs has started/ended.
//
// The idea behind a pool being specialized for a set of jobs is that some jobs need more
// time to complete and I don't want to hold back the other ones such as (in the future), sending
// emails, checking stuff, etc.
func NewPool(
	logger log.Logger,
	interval time.Duration,
	size int,
	handler func(context.Context, Runner) error,
	supportedJobs ...string,
) Worker {
	return &pool{
		logger:        logger,
		interval:      interval,
		supportedJobs: supportedJobs,
		size:          size,
		fn:            handler,
		done:          make(chan bool),
	}
}

func (p *pool) Start() {
	if p.started {
		p.logger.Warn("worker already started")
		return
	}

	p.started = true

	for {
		select {
		case <-p.done:
			p.exiting = true
			continue
		case <-time.After(p.interval): // TODO: maybe we must compute the fetch interval so that we can substract the time taken by the last job to the fetch interval to run another loop immediately
		}

		if p.exiting && len(p.currentJobs) == 0 {
			p.done <- true
			return
		}

		if p.exiting || len(p.currentJobs) >= p.size {
			continue
		}

		go p.work()
	}
}

func (p *pool) work() {
	if err := p.fn(context.Background(), p); err != nil {
		p.logger.Errorw("worker function failed",
			"error", err)
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

func (p *pool) Started(job string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentJobs = append(p.currentJobs, job)
}

func (p *pool) Ended(job string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, j := range p.currentJobs {
		if j == job {
			p.currentJobs = append(p.currentJobs[:i], p.currentJobs[i+1:]...)
			return
		}
	}
}

func (p *pool) SupportedJobs() []string { return p.supportedJobs }

func (p *pool) RunningJobs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentJobs
}
