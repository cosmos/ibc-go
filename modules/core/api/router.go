package api

import (
	"errors"
	"fmt"

	radix "github.com/armon/go-radix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Router contains all the module-defined callbacks required by IBC Protocol V2.
type Router struct {
	// routes is a radix trie that provides a prefix-based
	// look-up structure. It maps portIDs and their prefixes to
	// IBCModules.
	routes radix.Tree
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		routes: *radix.New(),
	}
}

// AddRoute registers a route for a given port ID prefix to a given IBCModule.
func (rtr *Router) AddRoute(portID string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(portID) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if rtr.HasRoute(portID) {
		panic(fmt.Errorf("route %s has already been registered", portID))
	}

	rtr.routes.Insert(portID, cbs)

	return rtr
}

// Route returns the IBCModule for a given portID.
func (rtr *Router) Route(portID string) IBCModule {
	_, route, ok := rtr.routes.LongestPrefix(portID)
	if !ok {
		panic(fmt.Sprintf("no route for %s", portID))
	}
	return route.(IBCModule)
}

// HasRoute returns true if the Router has a module registered for the portID or false otherwise.
func (rtr *Router) HasRoute(portID string) bool {
	_, ok := rtr.routes.Get(portID)
	return ok
}

// HasPrefixRoute returns true if the Router has a module registered for the given portID or its prefix.
// Returns false otherwise.
func (rtr *Router) HasPrefixRoute(portIDPrefix string) bool {
	_, _, ok := rtr.routes.LongestPrefix(portIDPrefix)
	return ok
}
