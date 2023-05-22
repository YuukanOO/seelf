package command

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
)

func ProcessNext(reader domain.JobsReader, writer domain.JobsWriter, handler domain.Handler) func(context.Context) error {
	return func(ctx context.Context) error {
		job, err := reader.GetNextPendingJob(ctx)

		// No job yet, nothing to do.
		if errors.Is(err, apperr.ErrNotFound) {
			return nil
		}

		if err != nil {
			return err
		}

		if err = handler.Process(ctx, job); err != nil {
			job.Failed(err)
		} else {
			job.Done()
		}

		return writer.Write(ctx, &job)
	}
}
