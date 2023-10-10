package command

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/async"
)

// Process the next pending job.
// Here the `async.Runner` is given as it because this is a usecase extremely tied
// to the infrastructure.
func ProcessNext(
	reader domain.JobsReader,
	writer domain.JobsWriter,
	handler domain.Handler,
) func(context.Context, async.Runner) error {
	return func(ctx context.Context, runner async.Runner) error {
		job, err := reader.GetNextPendingJob(
			ctx,
			runner.SupportedJobs(),
			runner.RunningJobs(),
		)

		// No job yet, nothing to do.
		if errors.Is(err, apperr.ErrNotFound) {
			return nil
		}

		if err != nil {
			return err
		}

		dedupeName := job.DedupeName()

		runner.Started(dedupeName)

		defer func() {
			runner.Ended(dedupeName)
		}()

		if err = handler.Process(ctx, job); err != nil {
			job.Failed(err)
		} else {
			job.Done()
		}

		return writer.Write(ctx, &job)
	}
}
