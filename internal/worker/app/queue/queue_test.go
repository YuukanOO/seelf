package queue_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/worker/app/queue"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Queue(t *testing.T) {
	sut := func(prepareErr error) bus.RequestHandler[string, queue.Command] {
		return queue.Handler(memory.NewJobsStore(), &dummyHandler{err: prepareErr})
	}

	t.Run("should returns an error if no handler can prepare the payload", func(t *testing.T) {
		prepareErr := errors.New("prepare_error")
		uc := sut(prepareErr)
		id, err := uc(context.Background(), queue.Command{
			Payload: "payload",
		})

		testutil.ErrorIs(t, prepareErr, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should queue a job", func(t *testing.T) {
		uc := sut(nil)
		id, err := uc(context.Background(), queue.Command{
			Payload: "payload",
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", id)
	})
}

type (
	dummyHandler struct {
		domain.Handler
		err error
	}

	payload struct {
		domain.JobData
	}
)

func (h *dummyHandler) Prepare(data any) (domain.JobData, monad.Maybe[string], error) {
	if h.err != nil {
		return nil, monad.None[string](), h.err
	}

	return payload{}, monad.None[string](), nil
}
