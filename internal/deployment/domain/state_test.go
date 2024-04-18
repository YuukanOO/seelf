package domain_test

import (
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_DeploymentState(t *testing.T) {
	t.Run("should be created in pending state", func(t *testing.T) {
		var state domain.DeploymentState

		testutil.Equals(t, domain.DeploymentStatusPending, state.Status())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsFalse(t, state.Services().HasValue())
		testutil.IsFalse(t, state.StartedAt().HasValue())
		testutil.IsFalse(t, state.FinishedAt().HasValue())
	})

	t.Run("could be marked as started", func(t *testing.T) {
		var (
			state domain.DeploymentState
			err   error
		)

		err = state.Started()

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentStatusRunning, state.Status())
		testutil.IsTrue(t, state.StartedAt().HasValue())
		testutil.IsFalse(t, state.FinishedAt().HasValue())
	})

	t.Run("could fail", func(t *testing.T) {
		var (
			state domain.DeploymentState
			err   error
		)

		testutil.IsNil(t, state.Started())

		err = state.Failed(errors.New("some error"))

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentStatusFailed, state.Status())
		testutil.Equals(t, "some error", state.ErrCode().MustGet())
		testutil.IsTrue(t, state.StartedAt().HasValue())
		testutil.IsTrue(t, state.FinishedAt().HasValue())
	})

	t.Run("could succeed", func(t *testing.T) {
		var (
			state    domain.DeploymentState
			err      error
			services domain.Services
		)

		app := must.Panic(domain.NewApp("app1",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("production-target"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("staging-target"), true, true),
			"uid"))
		conf := must.Panic(app.ConfigSnapshotFor(domain.Production))
		services, _ = services.Append(conf, "name1", "image1", false)
		testutil.IsNil(t, state.Started())

		err = state.Succeeded(services)

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, state.Status())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsTrue(t, state.Services().HasValue())
		testutil.DeepEquals(t, services, state.Services().MustGet())
		testutil.IsTrue(t, state.StartedAt().HasValue())
		testutil.IsTrue(t, state.FinishedAt().HasValue())
	})

	t.Run("should err if trying to start but not in pending state", func(t *testing.T) {
		var state domain.DeploymentState
		testutil.IsNil(t, state.Started())

		err := state.Started()

		testutil.ErrorIs(t, domain.ErrNotInPendingState, err)
	})

	t.Run("should err if trying to fail but not in runing state", func(t *testing.T) {
		var state domain.DeploymentState

		err := state.Failed(errors.New("an error"))

		testutil.ErrorIs(t, domain.ErrNotInRunningState, err)
	})

	t.Run("should err if trying to succeed but not in runing state", func(t *testing.T) {
		var state domain.DeploymentState

		err := state.Succeeded(domain.Services{})

		testutil.ErrorIs(t, domain.ErrNotInRunningState, err)
	})
}

func Test_TargetState(t *testing.T) {
	t.Run("should be created in configuring state", func(t *testing.T) {
		var state domain.TargetState

		testutil.Equals(t, domain.TargetStatusConfiguring, state.Status())
		testutil.IsTrue(t, state.Version().IsZero())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsFalse(t, state.LastReadyVersion().HasValue())
	})

	t.Run("can be reconfigured", func(t *testing.T) {
		var state domain.TargetState

		state.Reconfigure()

		testutil.Equals(t, domain.TargetStatusConfiguring, state.Status())
		testutil.IsFalse(t, state.Version().IsZero())
		testutil.IsFalse(t, state.ErrCode().HasValue())
	})

	t.Run("could be marked has done and sets the errcode and status appropriately", func(t *testing.T) {
		var (
			state     domain.TargetState
			errFailed = errors.New("failed")
		)
		state.Reconfigure()

		testutil.IsTrue(t, state.Configured(state.Version(), errFailed))

		testutil.Equals(t, domain.TargetStatusFailed, state.Status())
		testutil.Equals(t, errFailed.Error(), state.ErrCode().MustGet())
		testutil.IsFalse(t, state.LastReadyVersion().HasValue())

		state.Reconfigure()

		testutil.IsTrue(t, state.Configured(state.Version(), nil))
		testutil.Equals(t, state.Version(), state.LastReadyVersion().MustGet())

		testutil.Equals(t, domain.TargetStatusReady, state.Status())
		testutil.IsFalse(t, state.ErrCode().HasValue())
	})

	t.Run("should do nothing if the version does not match or if it has been already configured", func(t *testing.T) {
		var state domain.TargetState
		state.Reconfigure()

		testutil.IsFalse(t, state.Configured(state.Version().Add(-1), nil))

		testutil.Equals(t, domain.TargetStatusConfiguring, state.Status())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsFalse(t, state.Version().IsZero())
		testutil.IsFalse(t, state.LastReadyVersion().HasValue())

		state.Configured(state.Version(), nil)

		testutil.IsFalse(t, state.Configured(state.Version(), errors.New("should not happen")))

		testutil.Equals(t, domain.TargetStatusReady, state.Status())
		testutil.Equals(t, state.Version(), state.LastReadyVersion().MustGet())
		testutil.IsFalse(t, state.ErrCode().HasValue())
		testutil.IsFalse(t, state.Version().IsZero())
	})
}
