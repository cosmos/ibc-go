package api

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Router contains all the module-defined callbacks required by IBC Protocol V2.
type Router struct {
	// routes is a map associating port prefixes to the IBCModules implementations.
	routes map[string]IBCModule
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]IBCModule),
	}
}

// AddRoute registers a route for a given port ID prefix to a given IBCModule.
// Panics if a prefix of portIDprefix is already a registered route.
func (rtr *Router) AddRoute(portIDprefix string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(portIDprefix) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	prefixExists, prefix := rtr.HasRoute(portIDprefix)
	if prefixExists {
		panic(fmt.Errorf("route %s has already been covered by registered prefix: %s", portIDprefix, prefix))
	}

	rtr.routes[portIDprefix] = cbs

	return rtr
}

// Route returns the IBCModule for a given portID.
func (rtr *Router) Route(portID string) IBCModule {
	_, route, ok := rtr.getRoute(portID)
	if !ok {
		panic(fmt.Sprintf("no route for %s", portID))
	}
	return route
}

// HasRoute returns true along with a prefix if the router has a module
// registered for the given portID or its prefix. Returns false otherwise.
func (rtr *Router) HasRoute(portID string) (bool, string) {
	prefix, _, ok := rtr.getRoute(portID)
	return ok, prefix
}

func (rtr *Router) getRoute(portID string) (string, IBCModule, bool) {
	for prefix, module := range rtr.routes {
		if strings.HasPrefix(portID, prefix) {
			return prefix, module, true
		}
	}
	return "", nil, false
}
