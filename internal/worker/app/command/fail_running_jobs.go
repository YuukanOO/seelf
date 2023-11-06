package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/collections"
)

// Mark all running jobs as failed with the given reason. This is mostly used
// when the server has crashed or has been hard resetted and some job were processing.
func FailRunningJobs(
	reader domain.JobsReader,
	writer domain.JobsWriter,
) func(context.Context, error) error {
	return func(ctx context.Context, reason error) error {
		jobs, err := reader.GetRunningJobs(ctx)

		if err != nil {
			return err
		}

		for idx := range jobs {
			jobs[idx].Failed(reason)
		}

		return writer.Write(ctx, collections.ToPointers(jobs)...)
	}
}
