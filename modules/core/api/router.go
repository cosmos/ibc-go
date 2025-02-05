package api

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Router contains all the module-defined callbacks required by IBC Protocol V2.
type Router struct {
	// routes is a map from portID to IBCModule
	routes map[string]IBCModule
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]IBCModule),
	}
}

// AddRoute registers a route for a given portID to a given IBCModule.
func (rtr *Router) AddRoute(portID string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(portID) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if rtr.HasRoute(portID) {
		panic(fmt.Errorf("route %s has already been registered", portID))
	}

	rtr.routes[portID] = cbs

	return rtr
}

// Route returns the IBCModule for a given portID.
func (rtr *Router) Route(portID string) IBCModule {
	route, ok := rtr.routes[portID]
	if !ok {
		panic(fmt.Sprintf("no route for %s", portID))
	}
	return route
}

// HasRoute returns true if the Router has a module registered for the portID or false otherwise.
func (rtr *Router) HasRoute(portID string) bool {
	_, ok := rtr.routes[portID]
	return ok
}
