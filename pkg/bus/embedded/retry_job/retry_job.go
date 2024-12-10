package retry_job

import "github.com/YuukanOO/seelf/pkg/bus"

type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string { return "bus.command.retry_job" }
