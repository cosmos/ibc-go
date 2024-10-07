package api

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Router contains all the module-defined callbacks required by IBC Protocol V2.
type Router struct {
	routes map[string]IBCModule
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]IBCModule),
	}
}

// AddRoute registers a route for a given module name.
func (rtr *Router) AddRoute(module string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if rtr.HasRoute(module) {
		panic(fmt.Errorf("route %s has already been registered", module))
	}

	rtr.routes[module] = cbs

	return rtr
}

// Route returns the IBCModule for a given module name.
func (rtr *Router) Route(module string) IBCModule {
	route, ok := rtr.routeOrPrefixedRoute(module)
	if !ok {
		panic(fmt.Sprintf("no route for %s", module))
	}
	return route
}

// HasRoute returns true if the Router has a module registered or false otherwise.
func (rtr *Router) HasRoute(module string) bool {
	_, ok := rtr.routeOrPrefixedRoute(module)
	return ok
}

// routeOrPrefixedRoute returns the IBCModule for a given module name.
// if an exact match is not found, a route with the provided prefix is searched for instead.
func (rtr *Router) routeOrPrefixedRoute(module string) (IBCModule, bool) {
	route, ok := rtr.routes[module]
	if ok {
		return route, true
	}

	// it's possible that some routes have been dynamically added e.g. with interchain accounts.
	// in this case, we need to check if the module has the specified prefix.
	for prefix, ibcModule := range rtr.routes {
		if strings.HasPrefix(module, prefix) {
			return ibcModule, true
		}
	}
	return nil, false
}
