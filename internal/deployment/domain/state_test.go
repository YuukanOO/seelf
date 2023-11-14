package domain_test

import (
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_State(t *testing.T) {
	t.Run("should be created in pending state", func(t *testing.T) {
		var state domain.State

		testutil.Equals(t, domain.DeploymentStatusPending, state.Status())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsFalse(t, state.Services().HasValue())
		testutil.IsFalse(t, state.StartedAt().HasValue())
		testutil.IsFalse(t, state.FinishedAt().HasValue())
	})

	t.Run("could be marked as started", func(t *testing.T) {
		var (
			state domain.State
			err   error
		)

		state, err = state.Started()

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentStatusRunning, state.Status())
		testutil.IsTrue(t, state.StartedAt().HasValue())
		testutil.IsFalse(t, state.FinishedAt().HasValue())
	})

	t.Run("could fail", func(t *testing.T) {
		var (
			state domain.State
			err   error
		)

		state, _ = state.Started()

		state, err = state.Failed(errors.New("some error"))

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentStatusFailed, state.Status())
		testutil.Equals(t, "some error", state.ErrCode().MustGet())
		testutil.IsTrue(t, state.StartedAt().HasValue())
		testutil.IsTrue(t, state.FinishedAt().HasValue())
	})

	t.Run("could succeed", func(t *testing.T) {
		var (
			state    domain.State
			err      error
			services domain.Services
		)

		app := domain.NewApp("app1", "uid")
		conf := domain.NewConfig(app, domain.Production)
		services, _ = services.Internal(conf, "name1", "image1")
		state, _ = state.Started()

		state, err = state.Succeeded(services)

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, state.Status())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsTrue(t, state.Services().HasValue())
		testutil.DeepEquals(t, services, state.Services().MustGet())
		testutil.IsTrue(t, state.StartedAt().HasValue())
		testutil.IsTrue(t, state.FinishedAt().HasValue())
	})

	t.Run("should err if trying to start but not in pending state", func(t *testing.T) {
		state, _ := domain.State{}.Started()

		_, err := state.Started()

		testutil.ErrorIs(t, domain.ErrNotInPendingState, err)
	})

	t.Run("should err if trying to fail but not in runing state", func(t *testing.T) {
		var state domain.State

		_, err := state.Failed(errors.New("an error"))

		testutil.ErrorIs(t, domain.ErrNotInRunningState, err)
	})

	t.Run("should err if trying to succeed but not in runing state", func(t *testing.T) {
		var state domain.State

		_, err := state.Succeeded(domain.Services{})

		testutil.ErrorIs(t, domain.ErrNotInRunningState, err)
	})
}
