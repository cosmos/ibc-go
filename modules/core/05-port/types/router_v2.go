package types

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AppRouterV2 contains all the module-defined callbacks required by ICS-26
type AppRouterV2 struct {
	routes map[string]IBCModuleV2
}

func NewAppRouter() *AppRouterV2 {
	return &AppRouterV2{
		routes: make(map[string]IBCModuleV2),
	}
}

func (rtr *AppRouterV2) AddRoute(module string, cbs IBCModuleV2) *AppRouterV2 {
	if !sdk.IsAlphaNumeric(module) {
		panic(errors.New("route expressions can only contain alphanumeric characters"))
	}

	rtr.routes[module] = cbs

	return rtr
}

func (rtr *AppRouterV2) Route(appName string) IBCModuleV2 {
	route, ok := rtr.route(appName)
	if !ok {
		panic(fmt.Sprintf("no route for %s", appName))
	}

	return route
}

func (rtr *AppRouterV2) route(appName string) (IBCModuleV2, bool) {
	route, ok := rtr.routes[appName]
	if ok {
		return route, true
	}

	// it's possible that some routes have been dynamically added e.g. with interchain accounts.
	// in this case, we need to check if the module has the specified prefix.
	for prefix := range rtr.routes {
		if strings.HasPrefix(appName, prefix) {
			return rtr.routes[prefix], true
		}
	}
	return nil, false
}
