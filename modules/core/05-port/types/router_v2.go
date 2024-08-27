// NOTE: router_v2 is added to incrementally add functionality and switch over parts of codebase while leaving current router in place
// Eventually this will replace the v1 router.
package types

import (
	"errors"
	"fmt"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
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
	v2Routes     map[string]IBCModuleV2

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
		v2Routes:      make(map[string]IBCModuleV2),
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
func (rtr *AppRouter) AddRoute(module string, cbs ClassicIBCModule) *AppRouter {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	rtr.legacyRoutes[module] = cbs

	return rtr
}
func (rtr *AppRouter) AddV2Route(module string, cbs IBCModuleV2) *AppRouter {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	rtr.v2Routes[module] = cbs

	return rtr
}

// PacketRoute returns a list of callbacks. It takes the portID of the packet
// (used for classic IBC packets) and the packet data (used for multi-packetdata's).
// PacketRoute is explicitly seprated from the handshake route which only handles
// ClassicIBCModule's. Non ClassicIBCModule routing does not work on handshakes.
func (rtr *AppRouter) PacketRoute(packet channeltypes.PacketV2, module string) ([]IBCModule, bool) {
	if module == sentinelMultiPacketData {
		return rtr.routeMultiPacketData(packet)
	}
	legacyModule, ok := rtr.routeToLegacyModule(module)
	if !ok {
		return nil, false
	}
	return []IBCModule{legacyModule}, true
}

func (rtr *AppRouter) Route(appName string) IBCModuleV2 {
	route, ok := rtr.v2Routes[appName]
	if !ok {
		panic(fmt.Sprintf("no route for %s", appName))
	}

	return route
}

// TODO: docstring once implementation is complete
// https://github.com/cosmos/ibc-go/issues/7056
func (rtr *AppRouter) routeMultiPacketData(packetDataV2 channeltypes.PacketV2) ([]IBCModule, bool) {
	var cbs []IBCModule
	for _, pd := range packetDataV2.Data {
		route, ok := rtr.routes[pd.AppName]
		if !ok {
			panic(fmt.Sprintf("no route for %s", pd.AppName))
		}
		cbs = append(cbs, route)
	}
	return cbs, len(cbs) > 0
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
// as it is required only for those callbacks. It takes in the portID associated with the
// handshake.
func (rtr *AppRouter) HandshakeRoute(portID string) (ClassicIBCModule, bool) {
	route, ok := rtr.routeToLegacyModule(portID)
	return route, ok
}
