package types

import (
	"errors"
	"fmt"
	"strings"

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

func (rtr *AppRouter) Route(appName string) IBCModuleV2 {
	route, ok := rtr.route(appName)
	if !ok {
		panic(fmt.Sprintf("no route for %s", appName))
	}

	return route
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
