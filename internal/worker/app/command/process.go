package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
)

// Process a single job. Called by async workers that's why there are no command here.
func Process(writer domain.JobsWriter, handler domain.Handler) func(context.Context, domain.Job) error {
	return func(ctx context.Context, job domain.Job) error {
		if err := handler.Process(ctx, job); err != nil {
			job.Failed(err)
		} else {
			job.Done()
		}

		return writer.Write(ctx, &job)
	}
}
