package get_targets

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/get_target"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Retrieve all available targets.
type Query struct {
	bus.Query[[]get_target.Target]
}

func (Query) Name_() string { return "deployment.query.get_targets" }
