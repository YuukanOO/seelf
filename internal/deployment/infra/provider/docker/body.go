package docker

import "github.com/YuukanOO/seelf/pkg/monad"

// Request payload when wanting to instantiate a ProviderConfig
// If the Host field is omitted, the provider will consider it's a local target.
type Body struct {
	Host       monad.Maybe[string] `json:"host"`
	Port       monad.Maybe[int]    `json:"port"`
	User       monad.Maybe[string] `json:"user"`
	PrivateKey monad.Patch[string] `json:"private_key"`
}
