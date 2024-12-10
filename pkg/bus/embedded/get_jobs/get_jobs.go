package get_jobs

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	Job struct {
		ID          string              `json:"id"`
		Group       string              `json:"group"`
		MessageName string              `json:"message_name"`
		MessageData string              `json:"message_data"`
		QueuedAt    time.Time           `json:"queued_at"`
		NotBefore   time.Time           `json:"not_before"`
		ErrorCode   monad.Maybe[string] `json:"error_code"`
		Retrieved   bool                `json:"retrieved"`
	}

	Query struct {
		bus.Query[storage.Paginated[Job]]

		Page monad.Maybe[int] `form:"page"`
	}
)

func (q Query) Name_() string { return "bus.query.get_jobs" }
