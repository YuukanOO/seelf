package get_registries

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/get_registry"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Query struct {
	bus.Query[[]get_registry.Registry]
}

func (Query) Name_() string { return "deployment.query.get_registries" }
