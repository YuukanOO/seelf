package fail_running_jobs

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/collections"
)

// Mark all running jobs as failed with the given reason. This is mostly used
// when the server has crashed or has been hard resetted and some job were processing.
type Command struct {
	bus.Command[bool]

	Reason error `json:"-"`
}

func (Command) Name_() string { return "worker.command.fail_running_jobs" }

func Handler(
	reader domain.JobsReader,
	writer domain.JobsWriter,
) bus.RequestHandler[bool, Command] {
	return func(ctx context.Context, cmd Command) (bool, error) {
		jobs, err := reader.GetRunningJobs(ctx)

		if err != nil {
			return false, err
		}

		for idx := range jobs {
			jobs[idx].Failed(cmd.Reason)
		}

		if err = writer.Write(ctx, collections.ToPointers(jobs)...); err != nil {
			return false, err
		}

		return true, nil
	}
}
