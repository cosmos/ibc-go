package types

import (
	"errors"
	"fmt"
	"strings"

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
func (rtr *Router) Seal() {
	if rtr.sealed {
		panic(errors.New("router already sealed"))
	}
	rtr.sealed = true
}

// Sealed returns a boolean signifying if the Router is sealed or not.
func (rtr Router) Sealed() bool {
	return rtr.sealed
}

// AddRoute adds IBCModule for a given module name. It returns the Router
// so AddRoute calls can be linked. It will panic if the Router is sealed.
func (rtr *Router) AddRoute(module string, cbs IBCModule) *Router {
	if rtr.sealed {
		panic(fmt.Errorf("router sealed; cannot register %s route callbacks", module))
	}
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}
	if rtr.HasRoute(module) {
		panic(fmt.Errorf("route %s has already been registered", module))
	}

	rtr.routes[module] = cbs
	return rtr
}

// HasRoute returns true if the Router has a module registered or false otherwise.
func (rtr *Router) HasRoute(module string) bool {
	_, ok := rtr.routes[module]
	return ok
}

// GetRoute returns a IBCModule for a given module.
func (rtr *Router) GetRoute(module string) (IBCModule, bool) {
	route, ok := rtr.routes[module]
	if ok {
		return route, true
	}

	// it's possible that some routes have been dynamically added e.g. with interchain accounts.
	// in this case, we need to check if the module has the specified prefix.
	for prefix := range rtr.routes {
		if strings.Contains(module, prefix) {
			return rtr.routes[prefix], true
		}
	}
	return nil, false
}
