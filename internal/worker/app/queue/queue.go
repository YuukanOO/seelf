package queue

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Queue a new job which will be processed asynchronously.
type Command struct {
	bus.Command[string]

	Payload any `json:"payload"`
}

func (Command) Name_() string { return "worker.command.queue" }

func Handler(
	writer domain.JobsWriter,
	handler domain.Handler,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		data, dedupe, err := handler.Prepare(cmd.Payload)

		if err != nil {
			return "", err
		}

		job := domain.NewJob(data, dedupe)

		if err = writer.Write(ctx, &job); err != nil {
			return "", err
		}

		return string(job.ID()), nil
	}
}
