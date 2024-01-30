package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// The router is a map from module name to the LightClientModule
// which contains all the module-defined callbacks required by ICS-26
type Router struct {
	routes map[string]exported.LightClientModule
	sealed bool
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]exported.LightClientModule),
	}
}

// AddRoute adds LightClientModule for a given module name. It returns the Router
// so AddRoute calls can be linked. It will panic if the Router is sealed.
func (rtr *Router) AddRoute(module string, cbs exported.LightClientModule) *Router {
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

// GetRoute returns a LightClientModule for a given module.
func (rtr *Router) GetRoute(module string) (exported.LightClientModule, bool) {
	if !rtr.HasRoute(module) {
		return nil, false
	}
	return rtr.routes[module], true
}
