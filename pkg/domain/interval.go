package domain

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var ErrInvalidTimeInterval = apperr.New("invalid_time_interval")

type TimeInterval struct {
	from time.Time
	to   time.Time
}

// Builds up a new time interval.
func NewTimeInterval(from, to time.Time) (TimeInterval, error) {
	if from.After(to) {
		return TimeInterval{}, ErrInvalidTimeInterval
	}

	return TimeInterval{
		from: from,
		to:   to,
	}, nil
}

func (i TimeInterval) From() time.Time { return i.from }
func (i TimeInterval) To() time.Time   { return i.to }
