package api_login

import "github.com/YuukanOO/seelf/pkg/bus"

type Query struct {
	bus.Query[string]

	Key string
}

func (Query) Name_() string { return "auth.command.api_login" }

// Implemented directly by the gateway for now
