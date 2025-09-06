package types

import (
	"errors"
	"fmt"
	"sort"

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
func (rtr *Router) Sealed() bool {
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

// Route returns a IBCModule for a given module.
func (rtr *Router) Route(module string) (IBCModule, bool) {
	if !rtr.HasRoute(module) {
		return nil, false
	}
	return rtr.routes[module], true
}

// Keys returns the keys of the routes map.
func (rtr *Router) Keys() []string {
	keys := make([]string, 0, len(rtr.routes))

	for k := range rtr.routes {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

// GetMiddlewareStack returns the middleware stack for a given port.
// It returns the stack from bottom (base application) to top (last middleware).
// Note: This is a simplified implementation that returns the top-level module name.
// A complete implementation would require architecture changes to track the full stack.
func (rtr *Router) GetMiddlewareStack(portID string) []string {
	module, found := rtr.Route(portID)
	if !found {
		return nil
	}

	// For now, just return the top-level module name
	// This is a starting point that can be enhanced later
	return []string{module.ModuleName()}
}
