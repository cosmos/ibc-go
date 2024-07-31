// NOTE: router_v2 is added to incrementally add functionality and switch over parts of codebase while leaving current router in place
// Eventually this will replace the v1 router.
package types

import (
	"errors"
	"fmt"

	"golang.org/x/exp/maps"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AppRouter contains a map from module name to an ordered list of IBCModules
// which contains all the module-defined callbacks required by ICS-26.
type AppRouter struct {
	routes map[string][]ClassicIBCModule
	sealed bool
}

// NewAppRouter creates and returns a new IBCModule application router.
func NewAppRouter() *AppRouter {
	return &AppRouter{
		routes: make(map[string][]ClassicIBCModule),
	}
}

// Seal prevents the Router from any subsequent route handlers to be registered.
// Seal will panic if called more than once.
func (rtr *AppRouter) Seal() {
	if rtr.sealed {
		panic(errors.New("router already sealed"))
	}
	rtr.sealed = true
}

// Sealed returns a boolean signifying if the Router is sealed or not.
func (rtr AppRouter) Sealed() bool {
	return rtr.sealed
}

// AddRoute adds IBCModule for a given module name. It returns the Router
// so AddRoute calls can be linked. It will panic if the Router is sealed.
func (rtr *AppRouter) AddRoute(module string, cbs ClassicIBCModule) *AppRouter {
	if rtr.sealed {
		panic(fmt.Errorf("router sealed; cannot register %s route callbacks", module))
	}
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}
	rtr.routes[module] = append(rtr.routes[module], cbs)
	return rtr
}

// HasRoute returns true if the Router has a module registered or false otherwise.
func (rtr *AppRouter) HasRoute(module string) bool {
	_, ok := rtr.routes[module]
	return ok
}

// GetRoute returns a IBCModule for a given module.
func (rtr *AppRouter) GetRoute(module string) ([]ClassicIBCModule, bool) {
	if !rtr.HasRoute(module) {
		return nil, false
	}
	return rtr.routes[module], true
}

func (rtr *AppRouter) Keys() []string {
	return maps.Keys(rtr.routes)
}
