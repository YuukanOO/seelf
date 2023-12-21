package async

import (
	"context"
	"errors"
	"sync"

	"github.com/YuukanOO/seelf/pkg/log"
)

var ErrAlreadyStarted = errors.New("already_started")

type (
	// Represents an asynchronous pool which can process multiple jobs simultaneously.
	Pool[T any] interface {
		Start() error // Start the pool
		Stop()        // Wait for the current pool to complete running jobs and returns
	}

	Func func(<-chan bool) // Async function run by the pool

	// Pool context as seen by puller, groups and internal stuff.
	PoolContext[T any] interface {
		Queue(context.Context, ...T) // Queue one or multiple jobs to be processed
		Logger() log.Logger          // Returns the logger used by the pool
		Run(Func)                    // Launch a function in a new goroutine which MUST listen for the done signal and exit when it receives it
	}

	// Puller interface used to retrieve jobs to process from somewhere.
	Puller[T any] interface {
		OnStart(PoolContext[T]) // Called upon pool startup, use PoolContext.Run to register a goroutine
	}

	// Async group of runners which can actually handle jobs.
	Group[T any] interface {
		OnStart(PoolContext[T])         // Called upon pool startup, use PoolContext.Run to register a goroutine
		Handle(context.Context, T) bool // Try to handle the given job, returns true if the job was handled, false otherwise
	}

	pool[T any] struct {
		started   bool
		logger    log.Logger
		puller    Puller[T]
		done      []chan bool    // Contains exit channels used to signal participating functions to stop
		exitGroup sync.WaitGroup // Wait group used to wait for all workers to exit when stopping
		groups    []Group[T]
	}
)

// Builds a new pool with a specific Puller to feed registered groups with actual
// jobs to process.
//
// The idea behind groups is that some jobs need more time to complete and
// I don't want to hold back the other ones such as (in the future), sending
// emails, checking stuff, etc.
func NewPool[T any](
	logger log.Logger,
	puller Puller[T],
	groups ...Group[T],
) Pool[T] {
	return &pool[T]{
		logger: logger,
		puller: puller,
		groups: groups,
	}
}

func (p *pool[T]) Start() error {
	if p.started {
		return ErrAlreadyStarted
	}

	p.started = true

	for _, g := range p.groups {
		g.OnStart(p)
	}

	p.puller.OnStart(p)

	return nil
}

func (p *pool[T]) Stop() {
	if !p.started {
		return
	}

	p.logger.Info("waiting for current jobs to finish")

	for _, done := range p.done {
		done <- true
	}

	p.exitGroup.Wait()
}

func (p *pool[T]) Logger() log.Logger { return p.logger }

func (p *pool[T]) Run(fn Func) {
	done := make(chan bool, 1)
	p.done = append(p.done, done)

	p.exitGroup.Add(1)
	go func(d <-chan bool) {
		defer p.exitGroup.Done()
		fn(d)
	}(done)
}

func (p *pool[T]) Queue(ctx context.Context, jobs ...T) {
	for _, job := range jobs {
		for _, g := range p.groups {
			// Stop at the first group which can handle the job
			if g.Handle(ctx, job) {
				break
			}
		}
	}
}
