package api

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Router contains all the module-defined callbacks required by IBC Protocol V2.
type Router struct {
	// routes is a map from portID to IBCModule
	routes map[string]IBCModule
	// prefixRoutes is a map from portID prefix to IBCModule
	prefixRoutes map[string]IBCModule
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		routes:       make(map[string]IBCModule),
		prefixRoutes: make(map[string]IBCModule),
	}
}

// AddRoute registers a route for a given portID to a given IBCModule.
//
// Panics:
//   - if a route with the same portID has already been registered
//   - if the portID is not alphanumeric
func (rtr *Router) AddRoute(portID string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(portID) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if _, ok := rtr.routes[portID]; ok {
		panic(fmt.Errorf("route %s has already been registered", portID))
	}

	for prefix := range rtr.prefixRoutes {
		// Prevent existing prefix routes from colliding with the new direct route to avoid confusing behavior.
		if strings.HasPrefix(portID, prefix) {
			panic(fmt.Errorf("route %s is already matched by registered prefix route: %s", portID, prefix))
		}
	}

	rtr.routes[portID] = cbs

	return rtr
}

// AddPrefixRoute registers a route for a given portID prefix to a given IBCModule.
// A prefix route matches any portID that starts with the given prefix.
//
// Panics:
//   - if `portIDPrefix` is not alphanumeric.
//   - if a direct route `portIDPrefix` has already been registered.
//   - if a prefix of `portIDPrefix` is already registered as a prefix.
//   - if `portIDPrefix` is a prefix of am already registered prefix.
func (rtr *Router) AddPrefixRoute(portIDPrefix string, cbs IBCModule) *Router {
	if !sdk.IsAlphaNumeric(portIDPrefix) {
		panic(errors.New("route prefix can only contain alphanumeric characters"))
	}

	// If the prefix is a prefix of an already registered route, we panic to avoid confusing behavior.
	for portID := range rtr.routes {
		if strings.HasPrefix(portID, portIDPrefix) {
			panic(fmt.Errorf("route prefix %s is a prefix for already registered route: %s", portIDPrefix, portID))
		}
	}

	for prefix := range rtr.prefixRoutes {
		// Prevent two scenarios:
		//  * Adding a string that prefix is already registered e.g.
		//    add prefix "portPrefix" and try to add "portPrefixSomeSuffix".
		//  * Adding a string that is a prefix of already registered prefix route e.g.
		//    add prefix "portPrefix" and try to add "port".
		if strings.HasPrefix(portIDPrefix, prefix) {
			panic(fmt.Errorf("route prefix %s has already been covered by registered prefix: %s", portIDPrefix, prefix))
		}
		if strings.HasPrefix(prefix, portIDPrefix) {
			panic(fmt.Errorf("route prefix %s is a prefix for already registered prefix: %s", portIDPrefix, prefix))
		}
	}

	rtr.prefixRoutes[portIDPrefix] = cbs

	return rtr
}

// Route returns the IBCModule for a given portID.
func (rtr *Router) Route(portID string) IBCModule {
	cbs, ok := rtr.getRoute(portID)
	if !ok {
		panic(fmt.Sprintf("no route for %s", portID))
	}

	return cbs
}

// HasRoute returns true if the Router has a module registered (whether it's a direct or a prefix route)
// for the portID or false if no module is registered for it.
func (rtr *Router) HasRoute(portID string) bool {
	_, ok := rtr.getRoute(portID)
	return ok
}

// getRoute is a helper function that retrieves the IBCModule for a given portID.
func (rtr *Router) getRoute(portID string) (IBCModule, bool) {
	// Direct routes take precedence over prefix routes
	route, ok := rtr.routes[portID]
	if ok {
		return route, true
	}

	// If the portID is not found as a direct route, check for prefix routes
	for prefix, cbs := range rtr.prefixRoutes {
		// Note that this iteration is deterministic because there can only ever be one prefix route
		// that matches a given portID. This is because of the checks in AddPrefixRoute preventing
		// any colliding prefixes to be added.
		if strings.HasPrefix(portID, prefix) {
			return cbs, true
		}
	}

	// At this point neither a direct route nor a prefix route was found
	return nil, false
}
