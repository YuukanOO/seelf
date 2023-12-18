package async

import (
	"context"
)

type (
	HandlerFunc[T any]  func(context.Context, T) error // Process a single job
	SelectorFunc[T any] func(context.Context, T) bool  // Returns true if the group can handle the job

	group[T any] struct {
		jobs        chan T
		pool        Pool[T]
		handlerFunc HandlerFunc[T]
		canHandle   SelectorFunc[T]
		runners     []*runner[T]
	}

	runner[T any] struct {
		group *group[T]
	}
)

// Builds a new runners group which will process jobs satisfying the canHandle function.
func GroupFunc[T any](size int, handlerFunc HandlerFunc[T], canHandle SelectorFunc[T]) Group[T] {
	g := &group[T]{
		jobs:        make(chan T),
		handlerFunc: handlerFunc,
		canHandle:   canHandle,
		runners:     make([]*runner[T], size),
	}

	for i := 0; i < size; i++ {
		g.runners[i] = &runner[T]{g}
	}

	return g
}

func (g *group[T]) Handle(ctx context.Context, job T) bool {
	if g.canHandle(ctx, job) {
		g.jobs <- job
		return true
	}

	return false
}

func (g *group[T]) OnStart(pool Pool[T]) {
	g.pool = pool
	for _, r := range g.runners {
		pool.Participate(r.start)
	}
}

func (r *runner[T]) start(done <-chan bool) {
	for {
		select {
		case job := <-r.group.jobs:
			if err := r.group.handlerFunc(context.Background(), job); err != nil {
				r.group.pool.Logger().Errorw("error processing job",
					"error", err,
					"job", job)
			}
		case <-done:
			return
		}
	}
}
