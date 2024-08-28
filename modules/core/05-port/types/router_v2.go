package types

import (
	"errors"
	"fmt"
	"strings"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AppRouter contains all the module-defined callbacks required by ICS-26
type AppRouter struct {
	routes map[string]IBCModuleV2
}

func NewAppRouter() *AppRouter {
	return &AppRouter{
		routes: make(map[string]IBCModuleV2),
	}
}

func (rtr *AppRouter) AddV2Route(module string, cbs IBCModuleV2) *AppRouter {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	rtr.routes[module] = cbs

	return rtr
}

// Route returns a list of callbacks. It takes the packet data (used for multi-packetdata's).
func (rtr *AppRouter) Route(packet channeltypes.PacketV2) ([]IBCModuleV2, bool) {
	var cbs []IBCModuleV2
	for _, pd := range packet.Data {
		route, ok := rtr.route(pd.AppName)
		if !ok {
			panic(fmt.Sprintf("no route for %s", pd.AppName))
		}
		cbs = append(cbs, route)
	}
	return cbs, len(cbs) > 0
}

func (rtr *AppRouter) route(appName string) (IBCModuleV2, bool) {
	route, ok := rtr.routes[appName]
	if ok {
		return route, true
	}

	// it's possible that some routes have been dynamically added e.g. with interchain accounts.
	// in this case, we need to check if the module has the specified prefix.
	for prefix := range rtr.routes {
		if strings.Contains(appName, prefix) {
			return rtr.routes[prefix], true
		}
	}
	return nil, false
}
