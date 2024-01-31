package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	errorsmod "cosmossdk.io/errors"
)

func (k Keeper) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	// If the port capability is already claimed by a list of modules for the provided portID
	// then the modules must be identical to the ones passed in the message version for the message to be valid
	// For a non-routed version, the module owner of the capability is the one assumed to be opening the channel.
	application, portCap := k.LookupModuleByPort(ctx, portID)
	if portCap == nil {
		return "", errorsmod.Wrap(types.ErrInvalidPort, "port is not bound to a module")
	}
	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(version), &routedVersion)
	var modules []string
	if routeErr != nil {
		if routedVersion.Modules[len(routedVersion.Modules)-1] != application {
			return "", errorsmod.Wrap(types.ErrInvalidRoute, "port is already bound to a base IBC application that is different from expected")
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
	}

	chanCap, err := k.scopedKeeper.NewCapability(ctx, host.ChannelCapabilityPath(portID, channelID))
	if err != nil {
		return "", errorsmod.Wrapf(err, "could not create channel capability for port ID %s and channel ID %s", portID, channelID)
	}

	for i, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return "", errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}
		if routeErr != nil {
			version = routedVersion.Version[i]
		}

		version, err := cbs.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, routedVersion.Version[i])
		if routeErr != nil {
			return version, err
		}
		if err != nil {
			return "", err
		}
		routedVersion.Version[i] = version
	}
	return string(k.cdc.MustMarshalJSON(&routedVersion)), nil
}

func (k Keeper) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	// If the port capability is already claimed by a list of modules for the provided portID
	// then the modules must be identical to the ones passed in the message version for the message to be valid
	// For a non-routed version, the module owner of the capability is the one assumed to be opening the channel.
	application, portCap := k.LookupModuleByPort(ctx, portID)
	if portCap == nil {
		return "", errorsmod.Wrap(types.ErrInvalidPort, "port is not bound to a module")
	}
	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(counterpartyVersion), &routedVersion)
	var modules []string
	if routeErr != nil {
		if routedVersion.Modules[len(routedVersion.Modules)-1] != application {
			return "", errorsmod.Wrap(types.ErrInvalidRoute, "port is already bound to a base IBC application that is different from expected")
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
	}

	chanCap, err := k.scopedKeeper.NewCapability(ctx, host.ChannelCapabilityPath(portID, channelID))
	if err != nil {
		return "", errorsmod.Wrapf(err, "could not create channel capability for port ID %s and channel ID %s", portID, channelID)
	}

	for i, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return "", errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}
		if routeErr != nil {
			counterpartyVersion = routedVersion.Version[i]
		}

		version, err := cbs.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
		if routeErr != nil {
			return version, err
		}
		if err != nil {
			return "", err
		}
		routedVersion.Version[i] = version
	}
	return string(k.cdc.MustMarshalJSON(&routedVersion)), nil
}

func (k Keeper) OnChanOpenAck(ctx sdk.Context, portID, channelID, counterpartyChannelID, counterpartyVersion string) error {
	// Lookup module by channel capability
	application, _, err := k.LookupModuleByChannel(ctx, portID, channelID)
	if err != nil {
		ctx.Logger().Error("channel open ack failed", "port-id", portID, "error", errorsmod.Wrap(err, "could not retrieve module from port-id"))
		return errorsmod.Wrap(err, "could not retrieve module from port-id")
	}

	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(counterpartyVersion), &routedVersion)
	var modules []string
	if routeErr != nil {
		if routedVersion.Modules[len(routedVersion.Modules)-1] != application {
			return errorsmod.Wrap(types.ErrInvalidRoute, "port is already bound to a base IBC application that is different from expected")
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
	}

	for i, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}
		if routeErr != nil {
			counterpartyVersion = routedVersion.Version[i]
		}

		err := cbs.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	// Lookup module by channel capability
	application, _, err := k.LookupModuleByChannel(ctx, portID, channelID)
	if err != nil {
		ctx.Logger().Error("channel open ack failed", "port-id", portID, "error", errorsmod.Wrap(err, "could not retrieve module from port-id"))
		return errorsmod.Wrap(err, "could not retrieve module from port-id")
	}

	channel, ok := k.channelKeeper.GetChannel(ctx, portID, channelID)
	if !ok {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannel, "channel not found for portID (%s) channelID (%s)", portID, channelID)
	}
	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(channel.Version), &routedVersion)
	var modules []string
	if routeErr != nil {
		if routedVersion.Modules[len(routedVersion.Modules)-1] != application {
			return errorsmod.Wrap(types.ErrInvalidRoute, "port is already bound to a base IBC application that is different from expected")
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
	}

	for _, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}

		err := cbs.OnChanOpenConfirm(ctx, portID, channelID)
		if err != nil {
			return err
		}
	}
	return nil
}
