package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_Queue(t *testing.T) {
	queue := func() func(context.Context, command.QueueCommand) error {
		store := memory.NewJobsStore()
		return command.Queue(store)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := queue()
		err := uc(context.Background(), command.QueueCommand{})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should queue a job", func(t *testing.T) {
		uc := queue()
		err := uc(context.Background(), command.QueueCommand{
			Name:    "name",
			Payload: "payload",
		})

		testutil.IsNil(t, err)
	})
}
