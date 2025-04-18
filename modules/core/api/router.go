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
// Panics if a prefix of portIDprefix is already a registered route.
func (rtr *Router) AddRoute(portIDprefix string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(portIDprefix) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	prefixExists, prefix := rtr.HasRoute(portIDprefix)
	if prefixExists {
		panic(fmt.Errorf("route %s has already been covered by registered prefix: %s", portIDprefix, prefix))
	}

	rtr.routes.Insert(portIDprefix, cbs)

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

// HasPrefixRoute returns true if the Router has a module registered for the given portID or its prefix.
// Returns false otherwise.
func (rtr *Router) HasRoute(portID string) (bool, string) {
	prefix, _, ok := rtr.routes.LongestPrefix(portID)
	return ok, prefix
}
