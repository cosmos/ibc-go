package types

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// The router is a map from module name to the IBCModule
// which contains all the module-defined callbacks required by ICS-26
type Router struct {
	routes       map[string]IBCModule
	legacyRoutes map[string]ClassicIBCModule
	sealed       bool
}

func NewRouter() *Router {
	return &Router{
		routes:       make(map[string]IBCModule),
		legacyRoutes: make(map[string]ClassicIBCModule),
	}
}

// Seal prevents the Router from any subsequent route handlers to be registered.
// Seal will panic if called more than once.
func (rtr *Router) Seal() {
	if rtr.sealed {
		panic(errors.New("router already sealed"))
	}
	rtr.sealed = true
}

// Sealed returns a boolean signifying if the Router is sealed or not.
func (rtr Router) Sealed() bool {
	return rtr.sealed
}

// AddClassicRoute adds IBCModule for a given module name. It returns the Router
// so AddRoute calls can be linked. It will panic if the Router is sealed.
func (rtr *Router) AddClassicRoute(module string, cbs ...ClassicIBCModule) *Router {
	if rtr.sealed {
		panic(fmt.Errorf("router sealed; cannot register %s route callbacks", module))
	}
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	if _, ok := rtr.legacyRoutes[module]; ok {
		panic(fmt.Errorf("route %s has already been registered", module))
	}

	switch len(cbs) {
	case 0:
		panic(fmt.Errorf("no callbacks provided!"))
	default:
		rtr.legacyRoutes[module] = NewLegacyIBCModule(cbs...)
	}
	return rtr
}

// AddRoute adds IBCModule for a given module name. It returns the Router
// so AddRoute calls can be linked. It will panic if the Router is sealed.
func (rtr *Router) AddRoute(module string, cb IBCModule) *Router {
	if rtr.sealed {
		panic(fmt.Errorf("router sealed; cannot register %s route callbacks", module))
	}
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}
	if _, ok := rtr.routes[module]; ok {
		panic(fmt.Errorf("route %s has already been registered", module))
	}

	rtr.routes[module] = cb

	return rtr
}

func (rtr *Router) HandshakeRoute(portID string) (ClassicIBCModule, bool) {
	legacyRoute, ok := rtr.legacyRoutes[portID]
	return legacyRoute, ok
}

const sentinelMultiPacketData = "MultiPacketData"

func (rtr *Router) PacketRoute(packet channeltypes.Packet) ([]IBCModule, bool) {
	if packet.SourcePort == sentinelMultiPacketData {
		panic("unimplemented")
		// TODO: unimplemented
		//	for _, pd := range packet.Data {
		//      cbs = append(cbs, rtr.routes[pd.PortId])
		//  }
	}

	module := packet.SourcePort

	route, ok := rtr.legacyRoutes[module]
	if ok {
		return []IBCModule{route}, true
	}

	for prefix := range rtr.legacyRoutes {
		if strings.Contains(module, prefix) {
			return []IBCModule{rtr.legacyRoutes[prefix]}, true
		}
	}

	return nil, false
}

// Routes returns a IBCModule for a given module.
// TODO: return error instead of bool
//func (rtr *Router) Routes(packet channeltypes.Packet) ([]ClassicIBCModule, bool) {
//	if packet.SourcePort == "MultiPacketData" {
//		// TODO: unimplemented
//		//	for _, pd := range packet.Data {
//		//      cbs = append(cbs, rtr.routes[pd.PortId])
//		//  }
//	}
//
//	module := packet.SourcePort
//
//	route, ok := rtr.legacyRoutes[module]
//	if ok {
//		return []ClassicIBCModule{route}, true
//	}
//
//	for prefix := range rtr.routes {
//		if strings.Contains(module, prefix) {
//			return []ClassicIBCModule{rtr.legacyRoutes[prefix]}, true
//		}
//	}
//
//	return nil, false
//}
