package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/05-port/types"

	errorsmod "cosmossdk.io/errors"
)

func (k Keeper) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, portCap, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) (string, error) {
	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(version), &routedVersion)
	var routes []string
	if routeErr != nil {
		// do backward compatible onChanOpenInit to single module
		modules, _ := k.LookupModuleByPort(ctx, portID)
		if len(modules) != 1 {
			return "", errorsmod.Wrapf(types.ErrInvalidRoute, "expected single module bound to portID")
		}
		routes = []string{modules[0]}
	} else {
		for _, r := range routedVersion.Routes {
			routes = append(routes, r.Route)
		}
	}

	for i, route := range routes {
		cbs, exists := k.Router.GetRoute(route)
		if !exists {
			return "", errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", route)
		}

		version, err := cbs.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
		if routeErr != nil {
			return version, err
		}
		if err != nil {
			return "", err
		}
		routedVersion.Routes[i].Version.Version = version
	}
	return string(k.cdc.MustMarshalJSON(&routedVersion)), nil
}
