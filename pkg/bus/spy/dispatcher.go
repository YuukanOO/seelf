//go:build !release

package spy

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/bus"
)

type (
	Dispatcher interface {
		bus.Dispatcher

		Reset() // Clear all requests and signals
		Requests() []bus.Request
		Signals() []bus.Signal
	}

	dispatcher struct {
		requests []bus.Request
		signals  []bus.Signal
	}
)

// Builds a new dispatcher used for testing only. It will not send anything but
// append the requests and signals to the internal slices so they can be checked.
func NewDispatcher() Dispatcher {
	return &dispatcher{}
}

func (d *dispatcher) Send(ctx context.Context, msg bus.Request) (any, error) {
	d.requests = append(d.requests, msg)
	return nil, nil
}

func (d *dispatcher) Notify(ctx context.Context, msgs ...bus.Signal) error {
	d.signals = append(d.signals, msgs...)
	return nil
}

func (d *dispatcher) Reset() {
	d.requests = nil
	d.signals = nil
}

func (d *dispatcher) Requests() []bus.Request { return d.requests }
func (d *dispatcher) Signals() []bus.Signal   { return d.signals }
