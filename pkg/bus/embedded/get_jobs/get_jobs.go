package get_jobs

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	Job struct {
		Key         string    `json:"key"`
		Topic       string    `json:"topic"`
		MessageName string    `json:"message_name"`
		MessageData string    `json:"message_data"`
		QueuedAt    time.Time `json:"queued_at"`
		State       State     `json:"state"`
	}

	State struct {
		Status    uint8
		NotBefore time.Time
		ErrorCode monad.Maybe[string]
	}

	Query struct {
		bus.Query[storage.Paginated[Job]]

		Page monad.Maybe[int] `form:"page"`
	}
)

func (q Query) Name_() string { return "embedded.query.get_jobs" }
