package types

import (
	"fmt"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Router is a map from a clientType to a LightClientModule instance.
type Router struct {
	routes map[string]exported.LightClientModule
}

// NewRouter returns an instance of the Router.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]exported.LightClientModule),
	}
}

// AddRoute adds LightClientModule for a given module name. It returns the Router
// so AddRoute calls can be linked. This function will panic if:
// - the Router is sealed,
// - or a module is already registered for the provided client type,
// - or the client type is invalid.
func (rtr *Router) AddRoute(clientType string, module exported.LightClientModule) *Router {
	if rtr.HasRoute(clientType) {
		panic(fmt.Errorf("route %s has already been registered", module))
	}

	if err := ValidateClientType(clientType); err != nil {
		panic(fmt.Errorf("failed to add route: %w", err))
	}

	rtr.routes[clientType] = module
	return rtr
}

// HasRoute returns true if the Router has a module registered or false otherwise.
func (rtr *Router) HasRoute(clientType string) bool {
	_, ok := rtr.routes[clientType]
	return ok
}

// GetRoute returns the LightClientModule registered for the provided client type or false otherwise.
func (rtr *Router) GetRoute(clientType string) (exported.LightClientModule, bool) {
	if !rtr.HasRoute(clientType) {
		return nil, false
	}
	return rtr.routes[clientType], true
}
