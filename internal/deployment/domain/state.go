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
	State struct {
		status     DeploymentStatus
		errcode    monad.Maybe[string]
		services   monad.Maybe[Services]
		startedAt  monad.Maybe[time.Time]
		finishedAt monad.Maybe[time.Time]
	}
)

func (s State) Started() (State, error) {
	if s.status != DeploymentStatusPending {
		return s, ErrNotInPendingState
	}

	s.status = DeploymentStatusRunning
	s.startedAt.Set(time.Now().UTC())

	return s, nil
}

func (s State) Failed(err error) (State, error) {
	if s.status != DeploymentStatusRunning {
		return s, ErrNotInRunningState
	}

	s.status = DeploymentStatusFailed
	s.errcode.Set(err.Error())
	s.finishedAt.Set(time.Now().UTC())

	return s, nil
}

func (s State) Succeeded(services Services) (State, error) {
	if s.status != DeploymentStatusRunning {
		return s, ErrNotInRunningState
	}

	s.status = DeploymentStatusSucceeded
	s.services.Set(services)
	s.finishedAt.Set(time.Now().UTC())

	return s, nil
}

func (s State) Status() DeploymentStatus           { return s.status }
func (s State) ErrCode() monad.Maybe[string]       { return s.errcode }
func (s State) Services() monad.Maybe[Services]    { return s.services }
func (s State) StartedAt() monad.Maybe[time.Time]  { return s.startedAt }
func (s State) FinishedAt() monad.Maybe[time.Time] { return s.finishedAt }
