package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The router is a map from module name to the IBCModule
// which contains all the module-defined callbacks required by ICS-26
type Router struct {
	routes map[string]IBCModule
	sealed bool
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]IBCModule),
	}
}

// Seal prevents the Router from any subsequent route handlers to be registered.
// Seal will panic if called more than once.
func (rtr *Router) Seal() error {
	if rtr.sealed {
		return errors.New("router already sealed")
	}
	rtr.sealed = true
	return nil
}

// Sealed returns a boolean signifying if the Router is sealed or not.
func (rtr Router) Sealed() bool {
	return rtr.sealed
}

// AddRoute adds IBCModule for a given module name.
// It returns an error if the Router is sealed, the route contains
// non-alphanumeric characters or the route is already registered.
func (rtr *Router) AddRoute(module string, cbs IBCModule) error {
	if rtr.sealed {
		return fmt.Errorf("router sealed; cannot register %s route callbacks", module)
	}
	if !sdk.IsAlphaNumeric(module) {
		return errors.New("route expressions can only contain alphanumeric characters")
	}
	if rtr.HasRoute(module) {
		return fmt.Errorf("route %s has already been registered", module)
	}

	rtr.routes[module] = cbs
	return nil
}

// HasRoute returns true if the Router has a module registered or false otherwise.
func (rtr *Router) HasRoute(module string) bool {
	_, ok := rtr.routes[module]
	return ok
}

// GetRoute returns a IBCModule for a given module.
func (rtr *Router) GetRoute(module string) (IBCModule, bool) {
	if !rtr.HasRoute(module) {
		return nil, false
	}
	return rtr.routes[module], true
}
