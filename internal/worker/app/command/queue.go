package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type QueueCommand struct {
	Name       string              `json:"name"`
	Payload    string              `json:"payload"`
	DedupeName monad.Maybe[string] `json:"dedupe_name"`
}

func Queue(writer domain.JobsWriter) func(context.Context, QueueCommand) error {
	return func(ctx context.Context, cmd QueueCommand) error {
		if err := validation.Check(validation.Of{
			"name":        validation.Is(cmd.Name, strings.Required),
			"dedupe_name": validation.Maybe(cmd.DedupeName, strings.Required),
		}); err != nil {
			return err
		}

		job := domain.NewJob(cmd.Name, cmd.Payload, cmd.DedupeName)

		return writer.Write(ctx, &job)
	}
}
