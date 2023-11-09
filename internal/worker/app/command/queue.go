package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
)

type QueueCommand struct {
	Payload any `json:"payload"`
}

func Queue(
	writer domain.JobsWriter,
	handler domain.Handler,
) func(context.Context, QueueCommand) error {
	return func(ctx context.Context, cmd QueueCommand) error {
		data, dedupe, err := handler.Prepare(cmd.Payload)

		if err != nil {
			return err
		}

		job := domain.NewJob(data, dedupe)

		return writer.Write(ctx, &job)
	}
}
