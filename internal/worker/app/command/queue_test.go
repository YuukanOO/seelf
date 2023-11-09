package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/command"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Queue(t *testing.T) {
	queue := func(prepareErr error) func(context.Context, command.QueueCommand) error {
		return command.Queue(memory.NewJobsStore(), &dummyHandler{err: prepareErr})
	}

	t.Run("should returns an error if no handler can prepare the payload", func(t *testing.T) {
		prepareErr := errors.New("prepare_error")
		uc := queue(prepareErr)
		err := uc(context.Background(), command.QueueCommand{
			Payload: "payload",
		})

		testutil.ErrorIs(t, prepareErr, err)
	})

	t.Run("should queue a job", func(t *testing.T) {
		uc := queue(nil)
		err := uc(context.Background(), command.QueueCommand{
			Payload: "payload",
		})

		testutil.IsNil(t, err)
	})
}
