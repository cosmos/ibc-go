// NOTE: router_v2 is added to incrementally add functionality and switch over parts of codebase while leaving current router in place
// Eventually this will replace the v1 router.
package types

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: this is a temporary constant that is subject to change based on the final spec.
// https://github.com/cosmos/ibc/issues/1129
const sentinelMultiPacketData = "MultiPacketData"

// AppRouter contains all the module-defined callbacks required by ICS-26
type AppRouter struct {
	routes       map[string]IBCModule
	legacyRoutes map[string]ClassicIBCModule

	// classicRoutes facilitates the consecutive calls to AddRoute for existing modules.
	// TODO: this should be removed once app.gos have been refactored to use AddClassicRoute.
	// https://github.com/cosmos/ibc-go/issues/7025
	classicRoutes map[string][]ClassicIBCModule
}

func NewAppRouter() *AppRouter {
	return &AppRouter{
		routes:        make(map[string]IBCModule),
		legacyRoutes:  make(map[string]ClassicIBCModule),
		classicRoutes: make(map[string][]ClassicIBCModule),
	}
}

// AddClassicRoute takes a ordered list of ClassicIBCModules and creates a LegacyIBCModule. This is then added
// to the legacy mapping.
func (rtr *AppRouter) AddClassicRoute(module string, cbs ...ClassicIBCModule) *AppRouter {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if _, ok := rtr.legacyRoutes[module]; ok {
		panic(fmt.Errorf("route %s has already been registered", module))
	}

	if len(cbs) == 0 {
		panic(errors.New("no callbacks provided"))
	}

	rtr.legacyRoutes[module] = NewLegacyIBCModule(cbs...)

	return rtr
}

// AddRoute adds IBCModule for a given module name. It returns the Router
// so AddRoute calls can be linked. It will panic if the Router is sealed.
func (rtr *AppRouter) AddRoute(module string, cbs IBCModule) *AppRouter {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if classicModule, ok := cbs.(ClassicIBCModule); ok {
		rtr.classicRoutes[module] = append(rtr.classicRoutes[module], classicModule)

		// in order to facilitate having a single LegacyIBCModule, but also allowing for
		// consecutive calls to AddRoute to support existing functionality, we can re-create
		// the legacy module with the routes as they get added.
		if classicRoutes, ok := rtr.classicRoutes[module]; ok {
			rtr.legacyRoutes[module] = NewLegacyIBCModule(classicRoutes...)
		}
	} else {
		rtr.routes[module] = cbs
	}

	return rtr
}

func (rtr *AppRouter) PacketRoute(module string) ([]IBCModule, bool) {
	if module == sentinelMultiPacketData {
		return rtr.routeMultiPacketData(module)
	}
	legacyModule, ok := rtr.routeToLegacyModule(module)
	if !ok {
		return nil, false
	}
	return []IBCModule{legacyModule}, true
}

// TODO: docstring once implementation is complete
// https://github.com/cosmos/ibc-go/issues/7056
func (*AppRouter) routeMultiPacketData(module string) ([]IBCModule, bool) {
	panic("unimplemented")
	//  for _, pd := range packet.Data {
	//      cbs = append(cbs, rtr.routes[pd.PortId])
	//  }
	// return cbs, true
}

// routeToLegacyModule routes to any legacy modules which have been registered with AddClassicRoute.
func (rtr *AppRouter) routeToLegacyModule(module string) (ClassicIBCModule, bool) {
	route, ok := rtr.legacyRoutes[module]
	if ok {
		return route, true
	}

	// it's possible that some routes have been dynamically added e.g. with interchain accounts.
	// in this case, we need to check if the module has the specified prefix.
	for prefix := range rtr.legacyRoutes {
		if strings.Contains(module, prefix) {
			return rtr.legacyRoutes[prefix], true
		}
	}
	return nil, false
}

// HandshakeRoute returns the ClassicIBCModule which will implement all handshake functions
// and is required only for those callbacks.
func (rtr *AppRouter) HandshakeRoute(portID string) (ClassicIBCModule, bool) {
	route, ok := rtr.routeToLegacyModule(portID)
	return route, ok
}
