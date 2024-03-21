package domain

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
)

var (
	ErrNotInPendingState = apperr.New("not_in_pending_state")
	ErrNotInRunningState = apperr.New("not_in_running_state")
)

const (
	DeploymentStatusPending DeploymentStatus = iota
	DeploymentStatusRunning
	DeploymentStatusFailed
	DeploymentStatusSucceeded
)

type (
	DeploymentStatus uint8

	// Holds together information related to the current deployment state. With a value
	// object, it is easier to validate consistency between all those related properties.
	// The default value represents a pending state.
	DeploymentState struct {
		status     DeploymentStatus
		errcode    monad.Maybe[string]
		services   monad.Maybe[Services]
		startedAt  monad.Maybe[time.Time]
		finishedAt monad.Maybe[time.Time]
	}
)

func (s *DeploymentState) Started() error {
	if s.status != DeploymentStatusPending {
		return ErrNotInPendingState
	}

	s.status = DeploymentStatusRunning
	s.startedAt.Set(time.Now().UTC())

	return nil
}

func (s *DeploymentState) Failed(err error) error {
	if s.status != DeploymentStatusRunning {
		return ErrNotInRunningState
	}

	s.status = DeploymentStatusFailed
	s.errcode.Set(err.Error())
	s.finishedAt.Set(time.Now().UTC())

	return nil
}

func (s *DeploymentState) Succeeded(services Services) error {
	if s.status != DeploymentStatusRunning {
		return ErrNotInRunningState
	}

	s.status = DeploymentStatusSucceeded
	s.services.Set(services)
	s.finishedAt.Set(time.Now().UTC())

	return nil
}

func (s DeploymentState) Status() DeploymentStatus           { return s.status }
func (s DeploymentState) ErrCode() monad.Maybe[string]       { return s.errcode }
func (s DeploymentState) Services() monad.Maybe[Services]    { return s.services }
func (s DeploymentState) StartedAt() monad.Maybe[time.Time]  { return s.startedAt }
func (s DeploymentState) FinishedAt() monad.Maybe[time.Time] { return s.finishedAt }

const (
	TargetStatusConfiguring TargetStatus = iota
	TargetStatusFailed
	TargetStatusReady
)

type (
	TargetStatus uint8

	TargetState struct {
		status           TargetStatus
		version          time.Time
		errcode          monad.Maybe[string]
		lastReadyVersion monad.Maybe[time.Time] // Hold down the last time the target was marked as ready
	}
)

func newTargetState() (t TargetState) {
	t.Reconfigure()
	return t
}

// Mark the state as configuring and update the version.
func (t *TargetState) Reconfigure() {
	t.status = TargetStatusConfiguring
	t.version = time.Now().UTC()
	t.errcode.Unset()
}

// Update the state based on wether or not an error is given and returns a boolean indicating
// if the state has changed.
//
// If there is no error, the target will be considered ready.
// If an error is given, the target will be marked as failed.
//
// In either case, if the state has changed since it has been processed (the version param),
// it will return without doing anything because the result is outdated.
func (t *TargetState) Configured(version time.Time, err error) bool {
	if t.IsOutdated(version) {
		return false
	}

	if err != nil {
		t.status = TargetStatusFailed
		t.errcode.Set(err.Error())
		return true
	}

	t.status = TargetStatusReady
	t.lastReadyVersion.Set(version)
	t.errcode.Unset()

	return true
}

// Returns true if the given version is different from the current one or if the one
// provided is already configured.
func (t TargetState) IsOutdated(version time.Time) bool {
	return version != t.version || t.status != TargetStatusConfiguring
}

func (t TargetState) Status() TargetStatus                     { return t.status }
func (t TargetState) ErrCode() monad.Maybe[string]             { return t.errcode }
func (t TargetState) Version() time.Time                       { return t.version }
func (t TargetState) LastReadyVersion() monad.Maybe[time.Time] { return t.lastReadyVersion }
