package async

import (
	"context"
	"time"
)

type (
	PollerFunc[T any] func(context.Context) ([]T, error)

	intervalPuller[T any] struct {
		interval time.Duration
		poller   PollerFunc[T]
	}
)

// Simple puller wich will call the given function at the given interval.
func Poll[T any](interval time.Duration, pollFn PollerFunc[T]) Puller[T] {
	return &intervalPuller[T]{
		interval: interval,
		poller:   pollFn,
	}
}

func (p *intervalPuller[T]) OnStart(pool PoolContext[T]) {
	pool.Run(func(done <-chan bool) {
		var (
			delay   time.Duration
			lastRun time.Time = time.Now()
		)

		for {
			delay = p.interval - time.Since(lastRun)

			select {
			case <-done:
				return
			case <-time.After(delay):
			}

			lastRun = time.Now()
			ctx := context.Background()

			jobs, err := p.poller(ctx)

			if err != nil {
				pool.Logger().Errorw("error retrieving jobs",
					"error", err)
				continue
			}

			pool.Queue(ctx, jobs...)
		}
	})
}
