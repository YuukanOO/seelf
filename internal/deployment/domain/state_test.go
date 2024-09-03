package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_DeploymentState(t *testing.T) {
	t.Run("should be created in pending state", func(t *testing.T) {
		var state domain.DeploymentState

		assert.Equal(t, domain.DeploymentStatusPending, state.Status())
		assert.Zero(t, state.ErrCode())
		assert.False(t, state.Services().HasValue())
		assert.Zero(t, state.StartedAt())
		assert.Zero(t, state.FinishedAt())
	})

	t.Run("could be marked as started", func(t *testing.T) {
		var (
			state domain.DeploymentState
			err   error
		)

		err = state.Started()

		assert.Nil(t, err)
		assert.Equal(t, domain.DeploymentStatusRunning, state.Status())
		assert.Zero(t, state.ErrCode())
		assert.False(t, state.Services().HasValue())
		assert.NotZero(t, state.StartedAt())
		assert.Zero(t, state.FinishedAt())
	})

	t.Run("could fail", func(t *testing.T) {
		var (
			state    domain.DeploymentState
			givenErr = errors.New("some error")
			err      error
		)

		assert.Nil(t, state.Started())

		err = state.Failed(givenErr)

		assert.Nil(t, err)
		assert.Equal(t, domain.DeploymentStatusFailed, state.Status())
		assert.Equal(t, givenErr.Error(), state.ErrCode().Get(""))
		assert.False(t, state.Services().HasValue())
		assert.NotZero(t, state.StartedAt())
		assert.NotZero(t, state.FinishedAt())
	})

	t.Run("could succeed", func(t *testing.T) {
		var (
			state domain.DeploymentState
			err   error
		)

		deployment := fixture.Deployment()
		services := domain.Services{
			deployment.Config().NewService("app", ""),
		}
		assert.Nil(t, state.Started())

		err = state.Succeeded(services)

		assert.Nil(t, err)
		assert.Equal(t, domain.DeploymentStatusSucceeded, state.Status())
		assert.Zero(t, state.ErrCode())
		assert.DeepEqual(t, services, state.Services().Get(domain.Services{}))
		assert.NotZero(t, state.StartedAt())
		assert.NotZero(t, state.FinishedAt())
	})

	t.Run("should err if trying to start but not in pending state", func(t *testing.T) {
		var state domain.DeploymentState
		assert.Nil(t, state.Started())

		err := state.Started()

		assert.ErrorIs(t, domain.ErrNotInPendingState, err)
	})

	t.Run("should err if trying to fail but not in running state", func(t *testing.T) {
		var state domain.DeploymentState

		err := state.Failed(errors.New("an error"))

		assert.ErrorIs(t, domain.ErrNotInRunningState, err)
	})

	t.Run("should err if trying to succeed but not in running state", func(t *testing.T) {
		var state domain.DeploymentState

		err := state.Succeeded(domain.Services{})

		assert.ErrorIs(t, domain.ErrNotInRunningState, err)
	})
}

func Test_TargetState(t *testing.T) {
	t.Run("should be created in configuring state", func(t *testing.T) {
		var state domain.TargetState

		assert.Equal(t, domain.TargetStatusConfiguring, state.Status())
		assert.Zero(t, state.Version())
		assert.Zero(t, state.ErrCode())
		assert.Zero(t, state.LastReadyVersion())
	})

	t.Run("can be reconfigured", func(t *testing.T) {
		var state domain.TargetState

		state.Reconfigure()

		assert.Equal(t, domain.TargetStatusConfiguring, state.Status())
		assert.NotZero(t, state.Version())
		assert.Zero(t, state.ErrCode())
		assert.Zero(t, state.LastReadyVersion())
	})

	t.Run("could be marked has done and sets the errcode and status appropriately", func(t *testing.T) {
		var (
			state     domain.TargetState
			errFailed = errors.New("failed")
		)
		state.Reconfigure()

		assert.True(t, state.Configured(state.Version(), errFailed))

		assert.Equal(t, domain.TargetStatusFailed, state.Status())
		assert.NotZero(t, state.Version())
		assert.Equal(t, errFailed.Error(), state.ErrCode().Get(""))
		assert.Zero(t, state.LastReadyVersion())

		state.Reconfigure()

		assert.True(t, state.Configured(state.Version(), nil))
		assert.Equal(t, domain.TargetStatusReady, state.Status())
		assert.Equal(t, state.Version(), state.LastReadyVersion().Get(time.Time{}))
		assert.False(t, state.ErrCode().HasValue())
	})

	t.Run("should do nothing if the version does not match or if it has been already configured", func(t *testing.T) {
		var state domain.TargetState
		state.Reconfigure()

		assert.False(t, state.Configured(state.Version().Add(-1), nil))

		assert.Equal(t, domain.TargetStatusConfiguring, state.Status())
		assert.Zero(t, state.ErrCode())
		assert.Zero(t, state.LastReadyVersion())

		state.Configured(state.Version(), nil)

		assert.False(t, state.Configured(state.Version(), errors.New("should not happen")))

		assert.Equal(t, domain.TargetStatusReady, state.Status())
		assert.Equal(t, state.Version(), state.LastReadyVersion().Get(time.Time{}))
		assert.Zero(t, state.ErrCode())
	})
}
